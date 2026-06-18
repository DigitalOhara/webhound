package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/engine"
)

// testServer creates an httptest.Server that exposes a known structure.
func testServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Admin Panel</h1></body></html>"))
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"v1","status":"ok"}`))
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User-agent: *\nDisallow: /admin\n"))
	})
	mux.HandleFunc("/.env", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><body>Home</body></html>"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	})

	return httptest.NewServer(mux)
}

func TestScan_BasicDiscovery(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 5
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.StatusCodes = []int{200, 301, 302, 403}
	cfg.NoExtension = true
	cfg.Wordlists = []string{"common"}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestScan_RateLimitedScan(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 2
	cfg.Rate = 100
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"directories"}
	cfg.StatusCodes = []int{200, 403}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestScan_RecursiveMode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("<html><body>api</body></html>"))
	})
	mux.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"endpoint":"v1"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 5
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.Recursive = true
	cfg.MaxDepth = 2
	cfg.NoExtension = true
	cfg.Wordlists = []string{"directories"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestScan_ContextCancellation(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 2
	cfg.Rate = 1
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"common"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	_ = o.Run(ctx) // should not hang
}

func TestScan_WildcardSuppression(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Generic page content that is identical for all paths."))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.URL = srv.URL
	cfg.Threads = 5
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"directories"}
	cfg.StatusCodes = []int{200}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestScan_MultipleTargets(t *testing.T) {
	srv1 := testServer()
	srv2 := testServer()
	defer srv1.Close()
	defer srv2.Close()

	tmpFile, err := os.CreateTemp(t.TempDir(), "targets-*.txt")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(srv1.URL + "\n" + srv2.URL + "\n"); err != nil {
		t.Fatalf("writing targets file: %v", err)
	}
	tmpFile.Close()

	cfg := config.DefaultConfig()
	cfg.File = tmpFile.Name()
	cfg.Threads = 5
	cfg.Rate = 0
	cfg.Quiet = true
	cfg.NoConfirm = true
	cfg.NoExtension = true
	cfg.Wordlists = []string{"files"}
	cfg.StatusCodes = []int{200, 403}

	o, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
