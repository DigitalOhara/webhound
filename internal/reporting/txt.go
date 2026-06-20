package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

// WriteTXT writes scan results to a plain-text file, one result per line,
// in the same format shown in the terminal.
func WriteTXT(path string, cfg *config.ScanConfig, results []*response.Result, targets []string, startedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	// Header
	fmt.Fprintf(f, "WebHound v1.0.1 — Scan Report\n")
	fmt.Fprintf(f, "Started : %s\n", startedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(f, "Finished: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	for _, t := range targets {
		fmt.Fprintf(f, "Target  : %s\n", t)
	}
	fmt.Fprintf(f, "Results : %d\n", len(results))
	fmt.Fprintf(f, "%s\n\n", txtDivider)

	// Results
	for _, r := range results {
		fmt.Fprintln(f, formatTXTLine(r))
	}

	// Footer
	fmt.Fprintf(f, "\n%s\n", txtDivider)
	fmt.Fprintf(f, "Duration: %s\n", time.Since(startedAt).Round(time.Millisecond))

	_ = cfg // available for future use (e.g. printing scan config in header)
	return nil
}

const txtDivider = "────────────────────────────────────────────────────────────────────────"

func formatTXTLine(r *response.Result) string {
	size := formatBytes(r.ContentLength)
	rt := r.ResponseTime.Round(time.Millisecond)

	dir := ""
	if r.IsDirectory {
		dir = " [DIR]"
	}

	if r.RedirectURL != "" {
		return fmt.Sprintf("[%d] %-70s [%s] [%s]%s -> %s",
			r.StatusCode, r.URL, size, rt, dir, r.RedirectURL)
	}
	return fmt.Sprintf("[%d] %-70s [%s] [%s]%s",
		r.StatusCode, r.URL, size, rt, dir)
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
