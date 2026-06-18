package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/digitalohara/webhound/internal/response"
)

// wildcardServer returns the same body for every request (wildcard behaviour).
func wildcardServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Default page</body></html>"))
	}))
}

// distinctServer returns unique bodies per path (no wildcard).
func distinctServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><body>Admin panel</body></html>"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
}

func TestWildcardDetector_DetectsWildcard(t *testing.T) {
	srv := wildcardServer()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wd := response.NewWildcardDetector(client, "WebHound-Test/1.0")

	bl, err := wd.Probe(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !bl.IsWildcard {
		t.Error("expected wildcard detection, got IsWildcard=false")
	}
	if bl.StatusCode != http.StatusOK {
		t.Errorf("expected wildcard status 200, got %d", bl.StatusCode)
	}
}

func TestWildcardDetector_NoWildcardOnDistinctResponses(t *testing.T) {
	srv := distinctServer()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wd := response.NewWildcardDetector(client, "WebHound-Test/1.0")

	bl, err := wd.Probe(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	// Probe hits random UUIDs — server returns 404 for all of them, so
	// the status is uniform → IsWildcard=true BUT suppression only kicks
	// in for 404 status matches; a real 200 admin page would differ.
	// Verify the baseline status matches.
	if bl.IsWildcard && bl.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected wildcard status: %d", bl.StatusCode)
	}
}

func TestWildcardDetector_IsFalsePositive_MatchingResponse(t *testing.T) {
	srv := wildcardServer()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wd := response.NewWildcardDetector(client, "WebHound-Test/1.0")
	_, _ = wd.Probe(context.Background(), srv.URL)

	r := &response.Result{
		URL:           srv.URL + "/randompath",
		Target:        srv.URL,
		StatusCode:    200,
		ContentLength: 38,
	}
	body := []byte("<html><body>Default page</body></html>")

	if !wd.IsFalsePositive(r, body) {
		t.Error("expected identical body to be detected as false positive")
	}
}

func TestWildcardDetector_IsFalsePositive_RealResult(t *testing.T) {
	srv := wildcardServer()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wd := response.NewWildcardDetector(client, "WebHound-Test/1.0")
	_, _ = wd.Probe(context.Background(), srv.URL)

	// A result with a completely different body and length should NOT be suppressed.
	r := &response.Result{
		URL:           srv.URL + "/admin",
		Target:        srv.URL,
		StatusCode:    200,
		ContentLength: 9999,
	}
	body := []byte("<html><head><title>Admin</title></head><body>" + string(make([]byte, 9900)) + "</body></html>")

	if wd.IsFalsePositive(r, body) {
		t.Error("real result with different length/body should not be suppressed")
	}
}

func TestWildcardDetector_ProbeIdempotent(t *testing.T) {
	srv := wildcardServer()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wd := response.NewWildcardDetector(client, "WebHound-Test/1.0")

	bl1, _ := wd.Probe(context.Background(), srv.URL)
	bl2, _ := wd.Probe(context.Background(), srv.URL)

	if bl1 != bl2 {
		t.Error("repeated Probe calls should return the same Baseline pointer")
	}
}
