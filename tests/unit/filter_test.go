package unit_test

import (
	"testing"
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

func makeResult(code int, length int64, words, lines int) *response.Result {
	return &response.Result{
		URL:           "https://example.com/test",
		Path:          "test",
		StatusCode:    code,
		ContentLength: length,
		Words:         words,
		Lines:         lines,
		ResponseTime:  10 * time.Millisecond,
		FoundAt:       time.Now(),
	}
}

func TestFilter_AllowStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = []int{200, 301}
	f := response.NewFilter(cfg)

	if !f.Pass(makeResult(200, 100, 10, 5)) {
		t.Error("200 should pass allowlist [200,301]")
	}
	if !f.Pass(makeResult(301, 0, 0, 0)) {
		t.Error("301 should pass allowlist [200,301]")
	}
	if f.Pass(makeResult(404, 100, 10, 5)) {
		t.Error("404 should NOT pass allowlist [200,301]")
	}
}

func TestFilter_DenyStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = nil // no allowlist
	cfg.StatusCodesBlacklist = []int{404, 403}
	f := response.NewFilter(cfg)

	if f.Pass(makeResult(404, 100, 10, 5)) {
		t.Error("404 should be denied")
	}
	if f.Pass(makeResult(403, 100, 10, 5)) {
		t.Error("403 should be denied")
	}
	if !f.Pass(makeResult(200, 100, 10, 5)) {
		t.Error("200 should pass when not in denylist")
	}
}

func TestFilter_ContentLength(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = nil
	cfg.MinLength = 100
	cfg.MaxLength = 1000
	f := response.NewFilter(cfg)

	if f.Pass(makeResult(200, 50, 5, 2)) {
		t.Error("length 50 should be filtered (min=100)")
	}
	if f.Pass(makeResult(200, 2000, 200, 100)) {
		t.Error("length 2000 should be filtered (max=1000)")
	}
	if !f.Pass(makeResult(200, 500, 50, 20)) {
		t.Error("length 500 should pass (100-1000)")
	}
}

func TestFilter_HideLength(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = nil
	cfg.HideLength = []int{1234}
	f := response.NewFilter(cfg)

	if f.Pass(makeResult(200, 1234, 10, 5)) {
		t.Error("exact length 1234 should be hidden")
	}
	if !f.Pass(makeResult(200, 1235, 10, 5)) {
		t.Error("length 1235 should pass")
	}
}

func TestFilter_HideWords(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = nil
	cfg.HideWords = []int{42}
	f := response.NewFilter(cfg)

	if f.Pass(makeResult(200, 100, 42, 5)) {
		t.Error("42 words should be hidden")
	}
	if !f.Pass(makeResult(200, 100, 43, 5)) {
		t.Error("43 words should pass")
	}
}

func TestFilter_NoFilters(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.StatusCodes = nil
	f := response.NewFilter(cfg)

	for _, code := range []int{200, 301, 403, 404, 500} {
		if !f.Pass(makeResult(code, 100, 10, 5)) {
			t.Errorf("code %d should pass with no filters", code)
		}
	}
}
