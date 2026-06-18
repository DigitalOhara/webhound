package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

// JSONReport holds all data for a JSON report.
type JSONReport struct {
	Tool       string            `json:"tool"`
	Version    string            `json:"version"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	Duration   string            `json:"duration"`
	Config     reportConfig      `json:"config"`
	Results    []*response.Result `json:"results"`
	Statistics reportStats       `json:"statistics"`
}

type reportConfig struct {
	Targets    []string `json:"targets"`
	Threads    int      `json:"threads"`
	Rate       float64  `json:"rate"`
	MaxDepth   int      `json:"max_depth"`
	Extensions []string `json:"extensions"`
	AuthMode   string   `json:"auth_mode"`
}

type reportStats struct {
	TotalRequests int `json:"total_requests"`
	Found         int `json:"found"`
	Directories   int `json:"directories"`
	Files         int `json:"files"`
	Errors        int `json:"errors"`
}

// WriteJSON writes a JSON report to path.
func WriteJSON(path string, cfg *config.ScanConfig, results []*response.Result, targets []string, startedAt time.Time) error {
	stats := computeStats(results)
	report := JSONReport{
		Tool:       "webhound",
		Version:    "1.0.0",
		StartedAt:  startedAt,
		FinishedAt: time.Now(),
		Duration:   time.Since(startedAt).Round(time.Millisecond).String(),
		Config: reportConfig{
			Targets:    targets,
			Threads:    cfg.Threads,
			Rate:       cfg.Rate,
			MaxDepth:   cfg.MaxDepth,
			Extensions: cfg.Extensions,
			AuthMode:   detectAuthMode(cfg),
		},
		Results:    results,
		Statistics: stats,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON report: %w", err)
	}

	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("writing JSON report: %w", err)
	}
	return nil
}

func computeStats(results []*response.Result) reportStats {
	s := reportStats{Found: len(results)}
	for _, r := range results {
		if r.IsDirectory {
			s.Directories++
		} else {
			s.Files++
		}
	}
	return s
}

func detectAuthMode(cfg *config.ScanConfig) string {
	switch {
	case cfg.BearerToken != "":
		return "bearer"
	case cfg.BasicAuth != "":
		return "basic"
	case len(cfg.Cookies) > 0 || cfg.CookieFile != "":
		return "cookie"
	case cfg.Profile != "":
		return "profile:" + cfg.Profile
	default:
		return "none"
	}
}
