package ratelimit

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

const (
	maxBackoffSleep = 60 * time.Second
	baseBackoff     = 1 * time.Second
)

// BackoffState tracks retry state for a single request.
type BackoffState struct {
	Attempts int
	NextWait time.Duration
}

// NewBackoffState creates a fresh BackoffState.
func NewBackoffState() *BackoffState {
	return &BackoffState{NextWait: baseBackoff}
}

// ShouldRetry returns true if the response warrants a retry and increments
// the attempt counter.  It also sleeps for the calculated backoff duration.
func (b *BackoffState) ShouldRetry(ctx context.Context, resp *http.Response, maxRetries int) bool {
	if b.Attempts >= maxRetries {
		return false
	}
	if resp == nil {
		b.sleep(ctx)
		b.Attempts++
		return true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		wait := b.retryAfter(resp)
		b.Attempts++
		select {
		case <-time.After(wait):
		case <-ctx.Done():
		}
		return true
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusGatewayTimeout:
		b.sleep(ctx)
		b.Attempts++
		return true
	}
	return false
}

// retryAfter reads the Retry-After header; falls back to exponential backoff.
func (b *BackoffState) retryAfter(resp *http.Response) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.ParseFloat(ra, 64); err == nil {
			d := time.Duration(secs * float64(time.Second))
			if d < maxBackoffSleep {
				return d
			}
			return maxBackoffSleep
		}
	}
	return b.exponential()
}

func (b *BackoffState) sleep(ctx context.Context) {
	wait := b.exponential()
	select {
	case <-time.After(wait):
	case <-ctx.Done():
	}
}

func (b *BackoffState) exponential() time.Duration {
	d := b.NextWait
	b.NextWait *= 2
	if b.NextWait > maxBackoffSleep {
		b.NextWait = maxBackoffSleep
	}
	return d
}
