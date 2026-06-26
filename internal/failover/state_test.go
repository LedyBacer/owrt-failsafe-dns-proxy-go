package failover

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
)

func testManager() *Manager {
	return NewManager([]config.Upstream{
		{Name: "primary", Priority: 10},
		{Name: "backup", Priority: 20},
	}, 2, 2, nil)
}

func TestOperationalTransitionLogs(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	m := NewManager([]config.Upstream{
		{Name: "primary", Priority: 10, Address: "192.0.2.1", Port: 53},
		{Name: "backup", Priority: 20, Address: "192.0.2.2", Port: 53},
	}, 2, 2, logger)
	now := time.Unix(100, 0)
	m.RecordFailure("primary", "timeout", now)
	m.RecordFailure("primary", "timeout", now.Add(time.Second))
	m.RecordSuccess("primary", true, time.Millisecond, now.Add(2*time.Second))
	m.RecordSuccess("primary", true, time.Millisecond, now.Add(3*time.Second))

	logs := output.String()
	for _, expected := range []string{
		"upstream failure detected",
		"upstream unavailable",
		`msg="active upstream changed" from=primary to=backup`,
		"upstream recovery detected",
		"upstream recovered",
		`msg="active upstream changed" from=backup to=primary`,
	} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("missing log %q in:\n%s", expected, logs)
		}
	}
}

func TestEmergencyModeLogsOnlyOnChange(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	manager := NewManager([]config.Upstream{
		{Name: "primary", Priority: 10},
		{Name: "backup", Priority: 20},
	}, 1, 1, logger)

	manager.RecordSelectionMode(true)
	manager.RecordSelectionMode(true)
	manager.RecordSelectionMode(false)
	manager.RecordSelectionMode(false)

	logs := output.String()
	if strings.Count(logs, "emergency DNS path enabled") != 1 {
		t.Fatalf("expected one emergency enabled log, got %q", logs)
	}
	if strings.Count(logs, "emergency DNS path disabled") != 1 {
		t.Fatalf("expected one emergency disabled log, got %q", logs)
	}
}

func TestAttemptConcurrencyIsBounded(t *testing.T) {
	manager := testManager()
	var releases []func()
	for i := 0; i < 16; i++ {
		release, ok := manager.BeginAttempt("primary")
		if !ok {
			t.Fatalf("attempt %d was unexpectedly rejected", i)
		}
		releases = append(releases, release)
	}
	if _, ok := manager.BeginAttempt("primary"); ok {
		t.Fatal("17th concurrent attempt was accepted")
	}
	releases[0]()
	if release, ok := manager.BeginAttempt("primary"); !ok {
		t.Fatal("attempt was not accepted after release")
	} else {
		release()
	}
	for _, release := range releases[1:] {
		release()
	}
}

func TestManagerPreservesCompatibleSnapshotsOnReload(t *testing.T) {
	manager := testManager()
	now := time.Unix(100, 0)
	manager.RecordFailure("primary", "timeout", now)
	manager.RecordFailure("primary", "timeout", now.Add(time.Second))

	reloaded := NewManagerFromSnapshots([]config.Upstream{
		{Name: "primary", Priority: 5},
		{Name: "backup", Priority: 20},
	}, 3, 3, nil, manager.Snapshots())
	if got := reloaded.State("primary"); got != Down {
		t.Fatalf("compatible state not preserved: %s", got)
	}
	if got := reloaded.Snapshots()[0].Priority; got != 5 {
		t.Fatalf("new priority not applied: %d", got)
	}
}

func TestFailureAndRecoveryHysteresis(t *testing.T) {
	m := testManager()
	now := time.Unix(100, 0)
	m.RecordSuccess("primary", false, time.Millisecond, now)
	m.RecordFailure("primary", "timeout", now.Add(time.Second))
	if got := m.Snapshots()[0].State; got != Suspect {
		t.Fatalf("state after first failure = %s", got)
	}
	m.RecordFailure("primary", "timeout", now.Add(2*time.Second))
	if got := m.Snapshots()[0].State; got != Down {
		t.Fatalf("state after threshold = %s", got)
	}
	candidates, emergency := m.Candidates()
	if emergency || len(candidates) != 1 || candidates[0].Name != "backup" {
		t.Fatalf("unexpected candidates: %#v emergency=%v", candidates, emergency)
	}
	m.RecordSuccess("primary", true, time.Millisecond, now.Add(3*time.Second))
	if got := m.Snapshots()[0].State; got != Recovering {
		t.Fatalf("state after first recovery = %s", got)
	}
	m.RecordFailure("primary", "probe timeout", now.Add(3500*time.Millisecond))
	if got := m.Snapshots()[0].State; got != Down {
		t.Fatalf("failed recovery returned to %s instead of down", got)
	}
	m.RecordSuccess("primary", true, time.Millisecond, now.Add(3750*time.Millisecond))
	m.RecordSuccess("primary", true, time.Millisecond, now.Add(4*time.Second))
	if got := m.Snapshots()[0].State; got != Healthy {
		t.Fatalf("state after recovery threshold = %s", got)
	}
	if got := m.Active(); got != "primary" {
		t.Fatalf("active upstream = %q", got)
	}
}

func TestPassiveSuccessWhileDownDoesNotDeadlock(t *testing.T) {
	m := testManager()
	now := time.Unix(100, 0)
	m.RecordFailure("primary", "timeout", now)
	m.RecordFailure("primary", "timeout", now.Add(time.Second))

	m.RecordSuccess("primary", false, time.Millisecond, now.Add(2*time.Second))

	done := make(chan struct{})
	go func() {
		defer close(done)
		if got := m.Snapshots()[0].State; got != Down {
			t.Errorf("passive success changed down upstream to %s", got)
		}
		if got := m.Active(); got != "backup" {
			t.Errorf("active upstream = %q", got)
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("manager deadlocked after passive success on down upstream")
	}
}

func TestEmergencyHalfOpen(t *testing.T) {
	m := testManager()
	now := time.Now()
	for _, name := range []string{"primary", "backup"} {
		m.RecordFailure(name, "timeout", now)
		m.RecordFailure(name, "timeout", now)
	}
	candidates, emergency := m.Candidates()
	if !emergency || len(candidates) != 2 || candidates[0].Name != "primary" {
		t.Fatalf("unexpected emergency candidates: %#v emergency=%v", candidates, emergency)
	}
}

func TestConcurrentUpdates(t *testing.T) {
	m := testManager()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.RecordFailure("primary", "timeout", time.Now())
		}()
		go func() {
			defer wg.Done()
			m.RecordSuccess("primary", true, time.Millisecond, time.Now())
		}()
	}
	wg.Wait()
	_ = m.Snapshots()
	_, _ = m.Candidates()
}
