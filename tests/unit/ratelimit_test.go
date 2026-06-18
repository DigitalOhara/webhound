package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/digitalohara/webhound/internal/ratelimit"
)

func TestRateLimiter_Unlimited(t *testing.T) {
	mgr := ratelimit.NewManager(0, 0)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := mgr.Wait(ctx, "example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)
	// Unlimited should complete near-instantly.
	if elapsed > 500*time.Millisecond {
		t.Errorf("unlimited limiter took %v for 10 waits", elapsed)
	}
}

func TestRateLimiter_RateEnforced(t *testing.T) {
	// 10 RPS: 10 tokens in 1 second.
	mgr := ratelimit.NewManager(10, 0)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 11; i++ {
		if err := mgr.Wait(ctx, "target.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)
	// 11 requests at 10/s must take at least ~100ms.
	if elapsed < 80*time.Millisecond {
		t.Errorf("rate limiter too fast: %v for 11 requests at 10 RPS", elapsed)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	// 1 RPS: very slow.
	mgr := ratelimit.NewManager(1, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Drain the first token.
	_ = mgr.Wait(ctx, "slow.com")
	// Second wait should be cancelled.
	err := mgr.Wait(ctx, "slow.com")
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}

func TestRateLimiter_PerHostIsolation(t *testing.T) {
	// 2 RPS per host but two different hosts — should not block each other.
	mgr := ratelimit.NewManager(2, 0)
	ctx := context.Background()

	start := time.Now()
	done := make(chan struct{}, 2)

	go func() {
		for i := 0; i < 2; i++ {
			_ = mgr.Wait(ctx, "host-a.com")
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 2; i++ {
			_ = mgr.Wait(ctx, "host-b.com")
		}
		done <- struct{}{}
	}()
	<-done
	<-done

	elapsed := time.Since(start)
	// Two separate hosts × 2 requests each — should be fast in parallel.
	if elapsed > 2*time.Second {
		t.Errorf("per-host isolation broken: %v for 2+2 requests", elapsed)
	}
}

func TestBackoffState_MaxRetriesEnforced(t *testing.T) {
	bs := ratelimit.NewBackoffState()
	// A cancelled context causes ShouldRetry to skip the actual sleep,
	// letting us test the retry-count logic without waiting.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled — sleeps return immediately

	calls := 0
	for bs.ShouldRetry(ctx, nil, 3) {
		calls++
		if calls > 10 {
			t.Fatal("ShouldRetry looped too many times")
		}
	}
	if calls != 3 {
		t.Errorf("expected exactly 3 retries before stop, got %d", calls)
	}
}
