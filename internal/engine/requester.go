package engine

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/digitalohara/webhound/internal/auth"
	"github.com/digitalohara/webhound/internal/ratelimit"
	"github.com/digitalohara/webhound/internal/response"
	"github.com/digitalohara/webhound/pkg/utils"
)

// Requester executes a single HTTP job with rate limiting, auth, and retries.
type Requester struct {
	client     *http.Client
	authMgr    *auth.Manager
	rateMgr    *ratelimit.Manager
	userAgent  string
	maxRetries int
	method     string
}

// NewRequester creates a Requester.
func NewRequester(client *http.Client, authMgr *auth.Manager, rateMgr *ratelimit.Manager, userAgent string, maxRetries int, method string) *Requester {
	if method == "" {
		method = "GET"
	}
	return &Requester{
		client:     client,
		authMgr:    authMgr,
		rateMgr:    rateMgr,
		userAgent:  userAgent,
		maxRetries: maxRetries,
		method:     method,
	}
}

// Execute performs the HTTP request for job, applying rate limiting and retries.
func (r *Requester) Execute(ctx context.Context, job response.Job) *response.RawResult {
	host := utils.ExtractHost(job.BaseURL)
	backoff := ratelimit.NewBackoffState()

	targetURL := utils.JoinURL(job.BaseURL, job.Path)

	for {
		if err := r.rateMgr.Wait(ctx, host); err != nil {
			return &response.RawResult{Job: job, Error: err}
		}

		raw, shouldRetry := r.doOnce(ctx, job, targetURL, backoff)
		if !shouldRetry {
			return raw
		}
		if backoff.Attempts >= r.maxRetries {
			return raw
		}
	}
}

func (r *Requester) doOnce(ctx context.Context, job response.Job, targetURL string, backoff *ratelimit.BackoffState) (*response.RawResult, bool) {
	req, err := http.NewRequestWithContext(ctx, r.method, targetURL, nil)
	if err != nil {
		return &response.RawResult{Job: job, Error: err}, false
	}

	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	if r.authMgr != nil {
		r.authMgr.Apply(req)
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		raw := &response.RawResult{Job: job, Error: err, ResponseTime: elapsed}
		if backoff.ShouldRetry(ctx, nil, r.maxRetries) {
			return raw, true
		}
		return raw, false
	}
	defer resp.Body.Close()

	// Read up to 1 MB of body.
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	finalURL := resp.Request.URL.String()
	isRedirect := finalURL != targetURL && !strings.HasSuffix(finalURL, "/") == !strings.HasSuffix(targetURL, "/")

	raw := &response.RawResult{
		Job:           job,
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header,
		Body:          bodyBytes,
		ContentLength: resp.ContentLength,
		ResponseTime:  elapsed,
		FinalURL:      finalURL,
		IsRedirect:    isRedirect,
	}

	if backoff.ShouldRetry(ctx, resp, r.maxRetries) {
		return raw, true
	}
	return raw, false
}
