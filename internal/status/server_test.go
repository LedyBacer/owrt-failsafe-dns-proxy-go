package status

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/runtimecfg"
)

func TestStatusSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.sock")
	manager := failover.NewManager([]config.Upstream{
		{Name: "primary", Priority: 10, Protocol: "udp", Address: "192.0.2.1", Port: 53},
	}, 2, 2, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 1)
	go func() { errs <- New(path, "test", runtimecfg.New(config.Defaults(), manager)).Run(ctx) }()

	var doc Document
	var err error
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		doc, err = Read(path, 100*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if doc.Version != "test" || doc.Active != "primary" || len(doc.Upstreams) != 1 {
		t.Fatalf("unexpected status: %#v", doc)
	}
	if doc.Runtime.Goroutines < 1 || doc.Runtime.HeapAlloc == 0 {
		t.Fatalf("missing runtime metrics: %#v", doc.Runtime)
	}
	cancel()
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
}
