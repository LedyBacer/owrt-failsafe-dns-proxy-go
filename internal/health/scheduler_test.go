package health

import (
	"testing"
	"time"
)

func TestBackoffIsBoundedAndJittered(t *testing.T) {
	base := 5 * time.Second
	if got := backoff(base, 1, 0); got != 4*time.Second {
		t.Fatalf("first backoff = %v", got)
	}
	if got := backoff(base, 2, 1); got != 12*time.Second {
		t.Fatalf("second backoff = %v", got)
	}
	if got := backoff(base, 20, 0.5); got != 5*time.Minute {
		t.Fatalf("bounded backoff = %v", got)
	}
}
