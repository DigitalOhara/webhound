package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Manager maintains per-host token-bucket rate limiters plus a global limiter.
type Manager struct {
	mu       sync.Mutex
	hosts    map[string]*rate.Limiter
	global   *rate.Limiter
	rps      float64
	delay    time.Duration
	burst    int
}

// NewManager creates a rate limit manager.
// rps == 0 means unlimited (but per-host limiters can still be applied).
// delay is an extra fixed sleep added after each request per worker.
func NewManager(rps float64, delay time.Duration) *Manager {
	var global *rate.Limiter
	burst := 1
	if rps > 0 {
		burst = max(1, int(rps))
		global = rate.NewLimiter(rate.Limit(rps), burst)
	} else {
		global = rate.NewLimiter(rate.Inf, 1)
	}
	return &Manager{
		hosts:  make(map[string]*rate.Limiter),
		global: global,
		rps:    rps,
		delay:  delay,
		burst:  burst,
	}
}

// Wait blocks until a request to host is permitted by both the global
// and per-host limiters, or until ctx is cancelled.
func (m *Manager) Wait(ctx context.Context, host string) error {
	if err := m.global.Wait(ctx); err != nil {
		return err
	}
	hostLimiter := m.hostLimiter(host)
	if err := hostLimiter.Wait(ctx); err != nil {
		return err
	}
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// ThrottleHost temporarily reduces a host's rate to 1 RPS (called on 429/503).
func (m *Manager) ThrottleHost(host string, until time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Replace existing limiter with a 1-RPS limiter.
	m.hosts[host] = rate.NewLimiter(rate.Limit(1), 1)
}

// RestoreHost re-installs the original RPS limit for a host.
func (m *Manager) RestoreHost(host string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rps > 0 {
		m.hosts[host] = rate.NewLimiter(rate.Limit(m.rps), m.burst)
	} else {
		m.hosts[host] = rate.NewLimiter(rate.Inf, 1)
	}
}

func (m *Manager) hostLimiter(host string) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.hosts[host]; ok {
		return l
	}
	var l *rate.Limiter
	if m.rps > 0 {
		l = rate.NewLimiter(rate.Limit(m.rps), m.burst)
	} else {
		l = rate.NewLimiter(rate.Inf, 1)
	}
	m.hosts[host] = l
	return l
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
