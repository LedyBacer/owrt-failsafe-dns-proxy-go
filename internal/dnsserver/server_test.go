package dnsserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/runtimecfg"
)

type scriptedExchanger struct {
	calls []string
	fn    func(config.Upstream, *dns.Msg) (*dns.Msg, error)
}

func (s *scriptedExchanger) Exchange(_ context.Context, u config.Upstream, request *dns.Msg) (*dns.Msg, time.Duration, error) {
	s.calls = append(s.calls, u.Name)
	response, err := s.fn(u, request)
	return response, time.Millisecond, err
}

type recordingExchanger struct {
	mu    sync.Mutex
	calls []string
	seen  chan string
}

func (r *recordingExchanger) Exchange(_ context.Context, u config.Upstream, request *dns.Msg) (*dns.Msg, time.Duration, error) {
	r.mu.Lock()
	r.calls = append(r.calls, u.Name)
	r.mu.Unlock()
	r.seen <- u.Name
	response := new(dns.Msg)
	response.SetReply(request)
	return response, time.Millisecond, nil
}

func (r *recordingExchanger) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func testConfig() config.Config {
	cfg := config.Defaults()
	cfg.AttemptTimeout = 20 * time.Millisecond
	cfg.RequestTimeout = 100 * time.Millisecond
	cfg.Upstreams = []config.Upstream{
		{Name: "primary", Priority: 10, Protocol: "udp", Address: "192.0.2.1", Port: 53},
		{Name: "backup", Priority: 20, Protocol: "udp", Address: "192.0.2.2", Port: 53},
	}
	return cfg
}

func TestResolveFallsBackAndRemembersTransportFailure(t *testing.T) {
	cfg := testConfig()
	manager := failover.NewManager(cfg.Upstreams, 1, 2, nil)
	exchanger := &scriptedExchanger{fn: func(u config.Upstream, request *dns.Msg) (*dns.Msg, error) {
		if u.Name == "primary" {
			return nil, errors.New("timeout")
		}
		response := new(dns.Msg)
		response.SetReply(request)
		return response, nil
	}}
	server := New(runtimecfg.New(cfg, manager), exchanger, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	if _, err := server.Resolve(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(exchanger.calls) != 2 {
		t.Fatalf("calls = %#v", exchanger.calls)
	}
	exchanger.calls = nil
	if _, err := server.Resolve(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(exchanger.calls) != 1 || exchanger.calls[0] != "backup" {
		t.Fatalf("down upstream was retried: %#v", exchanger.calls)
	}
}

func TestResolveWaitsForPriorityAttemptSlot(t *testing.T) {
	cfg := testConfig()
	cfg.RequestTimeout = 250 * time.Millisecond
	manager := failover.NewManager(cfg.Upstreams, 1, 2, nil)
	var releases []func()
	for i := 0; i < 16; i++ {
		release, ok := manager.BeginAttempt("primary")
		if !ok {
			t.Fatalf("fill attempt %d was rejected", i)
		}
		releases = append(releases, release)
	}
	defer func() {
		for _, release := range releases {
			release()
		}
	}()

	exchanger := &recordingExchanger{seen: make(chan string, 1)}
	server := New(runtimecfg.New(cfg, manager), exchanger, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	done := make(chan error, 1)
	go func() {
		_, err := server.Resolve(context.Background(), request)
		done <- err
	}()

	select {
	case call := <-exchanger.seen:
		t.Fatalf("called %s before primary slot was available", call)
	case <-time.After(25 * time.Millisecond):
	}

	releases[0]()
	releases = releases[1:]
	select {
	case call := <-exchanger.seen:
		if call != "primary" {
			t.Fatalf("called %s after slot release, want primary", call)
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not use released primary slot")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if calls := exchanger.snapshot(); len(calls) != 1 || calls[0] != "primary" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestSERVFAILFallsBackWithoutPoisoningHealth(t *testing.T) {
	cfg := testConfig()
	manager := failover.NewManager(cfg.Upstreams, 1, 2, nil)
	exchanger := &scriptedExchanger{fn: func(u config.Upstream, request *dns.Msg) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.SetReply(request)
		if u.Name == "primary" {
			response.Rcode = dns.RcodeServerFailure
		}
		return response, nil
	}}
	server := New(runtimecfg.New(cfg, manager), exchanger, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	if _, err := server.Resolve(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if got := manager.Snapshots()[0].State; got != failover.Unknown {
		t.Fatalf("SERVFAIL changed global health to %s", got)
	}
}

func TestNXDOMAINIsSuccess(t *testing.T) {
	cfg := testConfig()
	manager := failover.NewManager(cfg.Upstreams, 2, 2, nil)
	exchanger := &scriptedExchanger{fn: func(_ config.Upstream, request *dns.Msg) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
		return response, nil
	}}
	server := New(runtimecfg.New(cfg, manager), exchanger, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := new(dns.Msg)
	request.SetQuestion("missing.example.", dns.TypeA)
	response, err := server.Resolve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Rcode != dns.RcodeNameError || manager.Snapshots()[0].State != failover.Healthy {
		t.Fatal("NXDOMAIN was not accepted as healthy response")
	}
}
