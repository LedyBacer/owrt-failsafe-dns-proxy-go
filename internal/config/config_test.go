package config

import (
	"strings"
	"testing"
	"time"
)

const validConfig = `
config main 'main'
	option enabled '1'
	option listen_addr '127.0.0.1'
	option listen_port '5359'
	option attempt_timeout_ms '100'
	option request_timeout_ms '250'
	option health_interval_s '3'
	option fail_threshold '2'
	option recover_threshold '3'
	option max_concurrent '16'
	option status_socket '/tmp/fdp.sock'
	list probe 'example.com:A'

config upstream 'backup'
	option priority '20'
	option protocol 'tcp'
	option address '192.0.2.2'
	option port '53'

config upstream 'primary'
	option priority '10'
	option protocol 'udp'
	option address '192.0.2.1'
	option port '5353'
`

func TestParseAndSort(t *testing.T) {
	cfg, err := Parse(strings.NewReader(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AttemptTimeout != 100*time.Millisecond || cfg.RequestTimeout != 250*time.Millisecond {
		t.Fatalf("unexpected timeouts: %v %v", cfg.AttemptTimeout, cfg.RequestTimeout)
	}
	if len(cfg.Upstreams) != 2 || cfg.Upstreams[0].Name != "primary" || cfg.Upstreams[1].Name != "backup" {
		t.Fatalf("upstreams not sorted: %#v", cfg.Upstreams)
	}
	if cfg.Probes[0].Name != "example.com." {
		t.Fatalf("probe was not normalized: %#v", cfg.Probes[0])
	}
}

func TestRejectHostnameAndDuplicatePriority(t *testing.T) {
	raw := strings.Replace(validConfig, "192.0.2.1", "resolver.example", 1)
	raw = strings.Replace(raw, "priority '20'", "priority '10'", 1)
	_, err := Parse(strings.NewReader(raw))
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	if !strings.Contains(message, "address must be an IP") || !strings.Contains(message, "duplicate upstream priority") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestBudgetMustCoverAttempt(t *testing.T) {
	raw := strings.Replace(validConfig, "request_timeout_ms '250'", "request_timeout_ms '50'", 1)
	_, err := Parse(strings.NewReader(raw))
	if err == nil || !strings.Contains(err.Error(), "request_timeout_ms") {
		t.Fatalf("expected timeout validation error, got %v", err)
	}
}
