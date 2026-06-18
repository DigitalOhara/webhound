package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/digitalohara/webhound/internal/config"
)

// Manager builds and injects authentication headers into every request.
type Manager struct {
	baseHeaders http.Header
}

// NewManager creates an auth Manager from the provided ScanConfig.
func NewManager(cfg *config.ScanConfig) (*Manager, error) {
	h := make(http.Header)

	// Bearer token
	if cfg.BearerToken != "" {
		h.Set("Authorization", "Bearer "+cfg.BearerToken)
	}

	// Basic auth
	if cfg.BasicAuth != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(cfg.BasicAuth))
		h.Set("Authorization", "Basic "+encoded)
	}

	// If both Bearer and Basic were given, bearer wins (set last wins)
	if cfg.BearerToken != "" && cfg.BasicAuth != "" {
		h.Set("Authorization", "Bearer "+cfg.BearerToken)
	}

	// Custom headers
	for _, header := range cfg.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q (expected 'Name: Value')", header)
		}
		h.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	// Cookies from strings
	allCookies := make([]string, 0, len(cfg.Cookies))
	allCookies = append(allCookies, cfg.Cookies...)

	// Cookies from file
	if cfg.CookieFile != "" {
		fileCookies, err := LoadCookieFile(cfg.CookieFile)
		if err != nil {
			return nil, fmt.Errorf("loading cookie file: %w", err)
		}
		allCookies = append(allCookies, fileCookies...)
	}

	if len(allCookies) > 0 {
		h.Set("Cookie", CookiesToHeader(allCookies))
	}

	// Apply profile if specified
	if cfg.Profile != "" {
		profile, err := config.LoadProfile(cfg.Profile)
		if err != nil {
			return nil, fmt.Errorf("loading profile %q: %w", cfg.Profile, err)
		}
		// Profile headers override individual flags
		if profile.BearerToken != "" {
			h.Set("Authorization", "Bearer "+profile.BearerToken)
		}
		if profile.BasicAuth != "" {
			h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(profile.BasicAuth)))
		}
		if len(profile.Cookies) > 0 {
			existing := h.Get("Cookie")
			extra := CookiesToHeader(profile.Cookies)
			if existing != "" {
				h.Set("Cookie", existing+"; "+extra)
			} else {
				h.Set("Cookie", extra)
			}
		}
		for k, v := range profile.Headers {
			h.Set(k, v)
		}
	}

	return &Manager{baseHeaders: h}, nil
}

// Apply injects the authentication headers into req.
// This is called for every outgoing request.
func (m *Manager) Apply(req *http.Request) {
	for k, values := range m.baseHeaders {
		for _, v := range values {
			req.Header.Set(k, v)
		}
	}
}

// HasAuth reports whether any authentication credentials are configured.
func (m *Manager) HasAuth() bool {
	return len(m.baseHeaders) > 0
}
