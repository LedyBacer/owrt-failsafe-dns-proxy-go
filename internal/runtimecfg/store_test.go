package runtimecfg

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
)

func TestUpdateIsAtomicAndTracksReload(t *testing.T) {
	first := config.Defaults()
	first.Upstreams = []config.Upstream{{Name: "first", Priority: 10}}
	store := New(first, failover.NewManager(first.Upstreams, 2, 2, nil))

	next := first
	next.RequestTimeout = 3 * time.Second
	next.Upstreams = []config.Upstream{{Name: "next", Priority: 20}}
	now := time.Unix(100, 0)
	store.Update(next, failover.NewManager(next.Upstreams, 3, 3, nil), now)

	got := store.Load()
	if got.Config.RequestTimeout != 3*time.Second || got.Manager.Active() != "next" {
		t.Fatalf("unexpected snapshot: %#v active=%q", got.Config, got.Manager.Active())
	}
	status := store.ReloadStatus()
	if status.Count != 1 || !status.LastAt.Equal(now) || status.LastError != "" {
		t.Fatalf("unexpected reload status: %#v", status)
	}
}

func TestValidateReloadRejectsListenerChanges(t *testing.T) {
	current := config.Defaults()
	next := current
	next.ListenPort++
	if err := ValidateReload(current, next); err == nil {
		t.Fatal("expected listener change error")
	}
	next = current
	next.RequestTimeout++
	if err := ValidateReload(current, next); err != nil {
		t.Fatalf("runtime-only change rejected: %v", err)
	}
}

func TestRecordReloadErrorKeepsCurrentSnapshot(t *testing.T) {
	cfg := config.Defaults()
	manager := failover.NewManager(nil, 2, 2, nil)
	store := New(cfg, manager)
	now := time.Unix(200, 0)
	store.RecordReloadError(errors.New("invalid config"), now)
	if store.Load().Manager != manager {
		t.Fatal("reload error replaced current snapshot")
	}
	status := store.ReloadStatus()
	if status.Count != 0 || status.LastError != "invalid config" || !status.LastAt.Equal(now) {
		t.Fatalf("unexpected reload error status: %#v", status)
	}
}
