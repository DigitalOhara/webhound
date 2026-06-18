package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/engine"
)

// authServer returns 200 only when the correct Authorization header is present.
func authServer(scheme, token string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		want := scheme + " " + token
		if auth != want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	}))
}

// cookieServer returns 200 only when the PHPSESSID cookie is present.
func cookieServer(cookieValue string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("PHPSESSID")
		if err != nil || cookie.Value != cookieValue {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cookie-authenticated"))
	}))
}

func TestAuth_BearerToken(t *testing.T) {
	const token = "test-bearer-token-12345"
	srv := authServer("Bearer", token)
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.BearerToken = token
	cfg.Threads = 2
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAuth_BasicAuth(t *testing.T) {
	srv := authServer("Basic", "dXNlcjpwYXNz") // user:pass base64
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.BasicAuth = "user:pass"
	cfg.Threads = 2
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAuth_CookieAuth(t *testing.T) {
	const sessionID = "abc123def456"
	srv := cookieServer(sessionID)
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Cookies = []string{"PHPSESSID=" + sessionID}
	cfg.Threads = 2
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAuth_CustomHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Headers = []string{"X-API-Key: secret-key"}
	cfg.Threads = 2
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestAuth_NoAuth_Returns401(t *testing.T) {
	const token = "required-token"
	srv := authServer("Bearer", token)
	defer srv.Close()

	// No auth configured — server will return 401 for all paths.
	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 2
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{401}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	// Should complete without error (401 is in the allowlist).
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
