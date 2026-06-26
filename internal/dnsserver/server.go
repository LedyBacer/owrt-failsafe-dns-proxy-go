package dnsserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/runtimecfg"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/upstream"
)

type Server struct {
	store     *runtimecfg.Store
	exchanger upstream.Exchanger
	logger    *slog.Logger
	udp       *dns.Server
	tcp       *dns.Server
}

func New(store *runtimecfg.Store, exchanger upstream.Exchanger, logger *slog.Logger) *Server {
	cfg := store.Load().Config
	s := &Server{
		store:     store,
		exchanger: exchanger,
		logger:    logger,
	}
	handler := dns.HandlerFunc(s.handle)
	addr := net.JoinHostPort(cfg.ListenAddr, fmt.Sprint(cfg.ListenPort))
	s.udp = &dns.Server{Addr: addr, Net: "udp", Handler: handler, UDPSize: 4096}
	s.tcp = &dns.Server{Addr: addr, Net: "tcp", Handler: handler}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	start := func(server *dns.Server, network string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Debug("DNS listener started", "network", network, "address", server.Addr)
			if err := server.ListenAndServe(); err != nil && ctx.Err() == nil {
				errs <- fmt.Errorf("%s listener: %w", network, err)
			}
		}()
	}
	start(s.udp, "udp")
	start(s.tcp, "tcp")

	select {
	case <-ctx.Done():
	case err := <-errs:
		_ = s.Shutdown()
		wg.Wait()
		return err
	}
	if err := s.Shutdown(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func (s *Server) Shutdown() error {
	var errs []error
	if err := s.udp.Shutdown(); err != nil && !strings.Contains(err.Error(), "server not started") {
		errs = append(errs, err)
	}
	if err := s.tcp.Shutdown(); err != nil && !strings.Contains(err.Error(), "server not started") {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s *Server) handle(w dns.ResponseWriter, request *dns.Msg) {
	runtime := s.store.Load()
	select {
	case runtime.Limit <- struct{}{}:
		defer func() { <-runtime.Limit }()
	default:
		_ = w.WriteMsg(failureResponse(request, dns.RcodeServerFailure))
		return
	}
	if len(request.Question) == 0 {
		_ = w.WriteMsg(failureResponse(request, dns.RcodeFormatError))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), runtime.Config.RequestTimeout)
	defer cancel()
	response, err := s.resolve(ctx, runtime, request)
	if err != nil {
		s.logger.Warn("DNS request failed", "error", err)
		_ = w.WriteMsg(failureResponse(request, dns.RcodeServerFailure))
		return
	}
	if err := w.WriteMsg(response); err != nil {
		s.logger.Debug("write DNS response", "error", err)
	}
}

func (s *Server) Resolve(ctx context.Context, request *dns.Msg) (*dns.Msg, error) {
	return s.resolve(ctx, s.store.Load(), request)
}

func (s *Server) resolve(ctx context.Context, runtime *runtimecfg.Snapshot, request *dns.Msg) (*dns.Msg, error) {
	candidates, emergency := runtime.Manager.Candidates()
	runtime.Manager.RecordSelectionMode(emergency)
	var failures []error
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		release, err := runtime.Manager.BeginAttemptContext(ctx, candidate.Name)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: wait for attempt slot: %w", candidate.Name, err))
			break
		}
		timeout := runtime.Config.AttemptTimeout
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				break
			}
			if remaining < timeout {
				timeout = remaining
			}
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		response, latency, err := s.exchanger.Exchange(attemptCtx, candidate, request)
		cancel()
		release()
		if err != nil {
			runtime.Manager.RecordFailure(candidate.Name, err.Error(), time.Now())
			failures = append(failures, fmt.Errorf("%s: %w", candidate.Name, err))
			continue
		}
		switch response.Rcode {
		case dns.RcodeServerFailure, dns.RcodeRefused:
			failures = append(failures, fmt.Errorf("%s returned %s", candidate.Name, dns.RcodeToString[response.Rcode]))
			continue
		default:
			runtime.Manager.RecordSuccess(candidate.Name, emergency, latency, time.Now())
			return response, nil
		}
	}
	return nil, errors.Join(failures...)
}

func failureResponse(request *dns.Msg, rcode int) *dns.Msg {
	response := new(dns.Msg)
	response.SetRcode(request, rcode)
	return response
}
