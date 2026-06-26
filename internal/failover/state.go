package failover

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
)

type State string

const (
	Unknown    State = "unknown"
	Healthy    State = "healthy"
	Suspect    State = "suspect"
	Down       State = "down"
	Recovering State = "recovering"
)

type Snapshot struct {
	Name                string        `json:"name"`
	Priority            int           `json:"priority"`
	Protocol            string        `json:"protocol"`
	Endpoint            string        `json:"endpoint"`
	State               State         `json:"state"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	ConsecutiveSuccess  int           `json:"consecutive_successes"`
	LastSuccess         time.Time     `json:"last_success,omitempty"`
	LastFailure         time.Time     `json:"last_failure,omitempty"`
	LastError           string        `json:"last_error,omitempty"`
	LastLatency         time.Duration `json:"last_latency_ns,omitempty"`
}

type entry struct {
	config   config.Upstream
	mu       sync.RWMutex
	snap     Snapshot
	attempts chan struct{}
}

type Manager struct {
	failThreshold    int
	recoverThreshold int
	entries          []*entry
	byName           map[string]*entry
	logger           *slog.Logger
	activeMu         sync.Mutex
	lastActive       string
	emergencyActive  bool
}

func NewManager(upstreams []config.Upstream, failThreshold, recoverThreshold int, logger *slog.Logger) *Manager {
	m := &Manager{
		failThreshold:    failThreshold,
		recoverThreshold: recoverThreshold,
		byName:           make(map[string]*entry, len(upstreams)),
		logger:           logger,
	}
	for _, upstream := range upstreams {
		e := &entry{
			config:   upstream,
			attempts: make(chan struct{}, 16),
			snap: Snapshot{
				Name:     upstream.Name,
				Priority: upstream.Priority,
				Protocol: upstream.Protocol,
				Endpoint: upstream.Endpoint(),
				State:    Unknown,
			},
		}
		m.entries = append(m.entries, e)
		m.byName[upstream.Name] = e
	}
	sort.SliceStable(m.entries, func(i, j int) bool {
		return m.entries[i].config.Priority < m.entries[j].config.Priority
	})
	if len(m.entries) > 0 {
		m.lastActive = m.entries[0].config.Name
	}
	return m
}

func NewManagerFromSnapshots(upstreams []config.Upstream, failThreshold, recoverThreshold int, logger *slog.Logger, previous []Snapshot) *Manager {
	manager := NewManager(upstreams, failThreshold, recoverThreshold, logger)
	byName := make(map[string]Snapshot, len(previous))
	for _, snapshot := range previous {
		byName[snapshot.Name] = snapshot
	}
	for _, e := range manager.entries {
		snapshot, ok := byName[e.config.Name]
		if !ok || snapshot.Endpoint != e.snap.Endpoint || snapshot.Protocol != e.snap.Protocol {
			continue
		}
		snapshot.Priority = e.snap.Priority
		e.snap = snapshot
	}
	manager.lastActive = manager.Active()
	return manager
}

func (m *Manager) BeginAttempt(name string) (func(), bool) {
	e := m.byName[name]
	if e == nil {
		return func() {}, false
	}
	select {
	case e.attempts <- struct{}{}:
		return func() { <-e.attempts }, true
	default:
		return func() {}, false
	}
}

func (m *Manager) BeginAttemptContext(ctx context.Context, name string) (func(), error) {
	e := m.byName[name]
	if e == nil {
		return func() {}, errors.New("unknown upstream")
	}
	select {
	case e.attempts <- struct{}{}:
		return func() { <-e.attempts }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

func (m *Manager) Candidates() (result []config.Upstream, emergency bool) {
	for _, e := range m.entries {
		e.mu.RLock()
		state := e.snap.State
		e.mu.RUnlock()
		if state != Down && state != Recovering {
			result = append(result, e.config)
		}
	}
	if len(result) > 0 {
		return result, false
	}
	for _, e := range m.entries {
		result = append(result, e.config)
	}
	return result, true
}

func (m *Manager) RecordSuccess(name string, probe bool, latency time.Duration, now time.Time) {
	e := m.byName[name]
	if e == nil {
		return
	}
	e.mu.Lock()
	oldState := e.snap.State
	changed := false
	e.snap.LastSuccess = now
	e.snap.LastLatency = latency
	e.snap.LastError = ""
	e.snap.ConsecutiveFailures = 0
	e.snap.ConsecutiveSuccess++
	switch e.snap.State {
	case Down, Recovering:
		if !probe {
			e.mu.Unlock()
			return
		}
		if e.snap.ConsecutiveSuccess >= m.recoverThreshold {
			e.snap.State = Healthy
		} else {
			e.snap.State = Recovering
		}
		changed = true
	default:
		e.snap.State = Healthy
		changed = true
	}
	newState := e.snap.State
	e.mu.Unlock()
	if changed {
		m.logTransition(e, oldState, newState)
	}
}

func (m *Manager) RecordFailure(name, message string, now time.Time) {
	e := m.byName[name]
	if e == nil {
		return
	}
	e.mu.Lock()
	oldState := e.snap.State
	e.snap.LastFailure = now
	e.snap.LastError = message
	e.snap.ConsecutiveSuccess = 0
	e.snap.ConsecutiveFailures++
	if e.snap.State == Down || e.snap.State == Recovering {
		e.snap.State = Down
		newState := e.snap.State
		e.mu.Unlock()
		m.logTransition(e, oldState, newState)
		return
	}
	if e.snap.ConsecutiveFailures >= m.failThreshold {
		e.snap.State = Down
		newState := e.snap.State
		e.mu.Unlock()
		m.logTransition(e, oldState, newState)
		return
	}
	e.snap.State = Suspect
	newState := e.snap.State
	e.mu.Unlock()
	m.logTransition(e, oldState, newState)
}

func (m *Manager) State(name string) State {
	e := m.byName[name]
	if e == nil {
		return Unknown
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.snap.State
}

func (m *Manager) Snapshots() []Snapshot {
	result := make([]Snapshot, 0, len(m.entries))
	for _, e := range m.entries {
		e.mu.RLock()
		result = append(result, e.snap)
		e.mu.RUnlock()
	}
	return result
}

func (m *Manager) Active() string {
	candidates, _ := m.Candidates()
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].Name
}

func (m *Manager) RecordSelectionMode(emergency bool) {
	if m.logger == nil {
		return
	}
	m.activeMu.Lock()
	defer m.activeMu.Unlock()
	if emergency == m.emergencyActive {
		return
	}
	m.emergencyActive = emergency
	if emergency {
		m.logger.Error("emergency DNS path enabled")
	} else {
		m.logger.Info("emergency DNS path disabled")
	}
}

func (m *Manager) logTransition(e *entry, oldState, newState State) {
	if m.logger == nil || oldState == newState {
		return
	}
	e.mu.RLock()
	snapshot := e.snap
	e.mu.RUnlock()
	switch newState {
	case Suspect:
		m.logger.Warn("upstream failure detected",
			"upstream", snapshot.Name,
			"endpoint", snapshot.Endpoint,
			"state", newState,
			"failures", snapshot.ConsecutiveFailures,
			"error", snapshot.LastError)
	case Down:
		m.logger.Error("upstream unavailable",
			"upstream", snapshot.Name,
			"endpoint", snapshot.Endpoint,
			"state", newState,
			"failures", snapshot.ConsecutiveFailures,
			"error", snapshot.LastError)
	case Recovering:
		m.logger.Info("upstream recovery detected",
			"upstream", snapshot.Name,
			"endpoint", snapshot.Endpoint,
			"state", newState,
			"successes", snapshot.ConsecutiveSuccess)
	case Healthy:
		if oldState == Recovering || oldState == Down {
			m.logger.Info("upstream recovered",
				"upstream", snapshot.Name,
				"endpoint", snapshot.Endpoint,
				"state", newState,
				"successes", snapshot.ConsecutiveSuccess)
		}
	}
	m.logActiveChange()
}

func (m *Manager) logActiveChange() {
	m.activeMu.Lock()
	defer m.activeMu.Unlock()
	active := m.Active()
	if active == "" || active == m.lastActive {
		return
	}
	previous := m.lastActive
	m.lastActive = active
	m.logger.Warn("active upstream changed", "from", previous, "to", active)
}
