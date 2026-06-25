package health

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/upstream"
)

type Scheduler struct {
	cfg       config.Config
	manager   *failover.Manager
	exchanger upstream.Exchanger
	logger    *slog.Logger
	rng       *rand.Rand
	rngMu     sync.Mutex
	dueMu     sync.Mutex
	due       map[string]time.Time
	failures  map[string]int
}

func New(cfg config.Config, manager *failover.Manager, exchanger upstream.Exchanger, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		manager:   manager,
		exchanger: exchanger,
		logger:    logger,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
		due:       make(map[string]time.Time),
		failures:  make(map[string]int),
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.probeAll(ctx)
	tick := s.cfg.HealthInterval / 4
	if tick < 250*time.Millisecond {
		tick = 250 * time.Millisecond
	}
	if tick > time.Second {
		tick = time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.probeAll(ctx)
		}
	}
}

func (s *Scheduler) probeAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, candidate := range s.cfg.Upstreams {
		candidate := candidate
		if !s.isDue(candidate.Name, time.Now()) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.probe(ctx, candidate)
		}()
	}
	wg.Wait()
}

func (s *Scheduler) probe(ctx context.Context, candidate config.Upstream) {
	probe := s.randomProbe()
	request := new(dns.Msg)
	request.SetQuestion(probe.Name, probe.QType)
	attemptCtx, cancel := context.WithTimeout(ctx, s.cfg.AttemptTimeout)
	defer cancel()
	response, latency, err := s.exchanger.Exchange(attemptCtx, candidate, request)
	now := time.Now()
	if err != nil {
		s.manager.RecordFailure(candidate.Name, "probe: "+err.Error(), now)
		s.schedule(candidate.Name, now, false)
		s.logger.Debug("upstream probe failed", "upstream", candidate.Name, "error", err)
		return
	}
	if response.Rcode == dns.RcodeServerFailure || response.Rcode == dns.RcodeRefused {
		s.manager.RecordFailure(candidate.Name, "probe: "+dns.RcodeToString[response.Rcode], now)
		s.schedule(candidate.Name, now, false)
		return
	}
	s.manager.RecordSuccess(candidate.Name, true, latency, now)
	s.schedule(candidate.Name, now, true)
}

func (s *Scheduler) randomProbe() config.Probe {
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.cfg.Probes[s.rng.Intn(len(s.cfg.Probes))]
}

func (s *Scheduler) isDue(name string, now time.Time) bool {
	s.dueMu.Lock()
	defer s.dueMu.Unlock()
	due, ok := s.due[name]
	if ok && now.Before(due) {
		return false
	}
	s.due[name] = now.Add(24 * time.Hour)
	return true
}

func (s *Scheduler) schedule(name string, now time.Time, success bool) {
	s.dueMu.Lock()
	defer s.dueMu.Unlock()
	if success {
		s.failures[name] = 0
		s.due[name] = now.Add(s.cfg.HealthInterval)
		return
	}
	if s.manager.State(name) == failover.Down {
		s.failures[name]++
	} else {
		s.failures[name] = 1
	}
	s.rngMu.Lock()
	random := s.rng.Float64()
	s.rngMu.Unlock()
	s.due[name] = now.Add(backoff(s.cfg.HealthInterval, s.failures[name], random))
}

func backoff(base time.Duration, failures int, random float64) time.Duration {
	if failures < 1 {
		failures = 1
	}
	delay := base
	for i := 1; i < failures && delay < 5*time.Minute; i++ {
		delay *= 2
	}
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	jitter := 0.8 + 0.4*random
	return time.Duration(float64(delay) * jitter)
}
