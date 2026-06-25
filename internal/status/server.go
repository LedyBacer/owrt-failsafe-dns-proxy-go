package status

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/runtimecfg"
)

type Document struct {
	Version   string                  `json:"version"`
	StartedAt time.Time               `json:"started_at"`
	Active    string                  `json:"active_upstream"`
	Upstreams []failover.Snapshot     `json:"upstreams"`
	Runtime   RuntimeStats            `json:"runtime"`
	Reload    runtimecfg.ReloadStatus `json:"reload"`
}

type RuntimeStats struct {
	Goroutines int    `json:"goroutines"`
	OpenFDs    int    `json:"open_fds"`
	HeapAlloc  uint64 `json:"heap_alloc_bytes"`
	HeapInUse  uint64 `json:"heap_inuse_bytes"`
	Sys        uint64 `json:"sys_bytes"`
}

type Server struct {
	path      string
	version   string
	startedAt time.Time
	store     *runtimecfg.Store
}

func New(path, version string, store *runtimecfg.Store) *Server {
	return &Server{path: path, version: version, startedAt: time.Now(), store: store}
}

func (s *Server) Run(ctx context.Context) error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale status socket: %w", err)
	}
	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return fmt.Errorf("listen on status socket: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(s.path)
	}()
	if err := os.Chmod(s.path, 0o660); err != nil {
		return fmt.Errorf("chmod status socket: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept status connection: %w", acceptErr)
		}
		go s.write(conn)
	}
}

func (s *Server) write(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	current := s.store.Load()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	openFDs := -1
	if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
		openFDs = len(entries)
	}
	_ = json.NewEncoder(conn).Encode(Document{
		Version:   s.version,
		StartedAt: s.startedAt,
		Active:    current.Manager.Active(),
		Upstreams: current.Manager.Snapshots(),
		Runtime: RuntimeStats{
			Goroutines: runtime.NumGoroutine(),
			OpenFDs:    openFDs,
			HeapAlloc:  memory.HeapAlloc,
			HeapInUse:  memory.HeapInuse,
			Sys:        memory.Sys,
		},
		Reload: s.store.ReloadStatus(),
	})
}

func Read(path string, timeout time.Duration) (Document, error) {
	conn, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return Document{}, fmt.Errorf("connect status socket: %w", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var doc Document
	if err := json.NewDecoder(io.LimitReader(conn, 1<<20)).Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("read status: %w", err)
	}
	return doc, nil
}
