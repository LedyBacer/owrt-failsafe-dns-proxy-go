package runtimecfg

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
)

type Snapshot struct {
	Config  config.Config
	Manager *failover.Manager
	Limit   chan struct{}
}

type ReloadStatus struct {
	Count     uint64    `json:"count"`
	LastAt    time.Time `json:"last_at,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

type Store struct {
	current atomic.Pointer[Snapshot]
	count   atomic.Uint64
	metaMu  sync.RWMutex
	lastAt  time.Time
	lastErr string
}

func New(cfg config.Config, manager *failover.Manager) *Store {
	store := &Store{}
	store.current.Store(newSnapshot(cfg, manager))
	return store
}

func (s *Store) Load() *Snapshot {
	return s.current.Load()
}

func (s *Store) Update(cfg config.Config, manager *failover.Manager, now time.Time) {
	s.current.Store(newSnapshot(cfg, manager))
	s.count.Add(1)
	s.metaMu.Lock()
	s.lastAt = now
	s.lastErr = ""
	s.metaMu.Unlock()
}

func (s *Store) RecordReloadError(err error, now time.Time) {
	s.metaMu.Lock()
	s.lastAt = now
	s.lastErr = err.Error()
	s.metaMu.Unlock()
}

func (s *Store) ReloadStatus() ReloadStatus {
	s.metaMu.RLock()
	defer s.metaMu.RUnlock()
	return ReloadStatus{
		Count:     s.count.Load(),
		LastAt:    s.lastAt,
		LastError: s.lastErr,
	}
}

func ValidateReload(current, next config.Config) error {
	var problems []error
	if current.ListenAddr != next.ListenAddr || current.ListenPort != next.ListenPort {
		problems = append(problems, fmt.Errorf(
			"listen address change from %s:%d to %s:%d requires restart",
			current.ListenAddr, current.ListenPort, next.ListenAddr, next.ListenPort,
		))
	}
	if current.StatusSocket != next.StatusSocket {
		problems = append(problems, errors.New("status_socket change requires restart"))
	}
	return errors.Join(problems...)
}

func newSnapshot(cfg config.Config, manager *failover.Manager) *Snapshot {
	return &Snapshot{
		Config:  cfg,
		Manager: manager,
		Limit:   make(chan struct{}, cfg.MaxConcurrent),
	}
}
