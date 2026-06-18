package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/digitalohara/webhound/internal/config"
)

// Manager loads and validates target URLs from config.
type Manager struct {
	cfg *config.ScanConfig
}

// NewManager creates a new input Manager.
func NewManager(cfg *config.ScanConfig) *Manager {
	return &Manager{cfg: cfg}
}

// LoadTargets returns the validated list of target base URLs.
func (m *Manager) LoadTargets() ([]string, error) {
	var raw []string

	switch {
	case m.cfg.URL != "":
		raw = []string{m.cfg.URL}
	case m.cfg.File != "":
		lines, err := readLines(m.cfg.File)
		if err != nil {
			return nil, fmt.Errorf("reading targets file %q: %w", m.cfg.File, err)
		}
		raw = lines
	default:
		return nil, fmt.Errorf("no target specified")
	}

	valid, errs := ValidateURLList(raw)
	for _, e := range errs {
		// non-fatal — log-worthy but don't abort
		_, _ = fmt.Fprintf(os.Stderr, "[WARN] %v\n", e)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid targets found")
	}
	return valid, nil
}

// readLines reads a file and returns non-empty, non-comment lines.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}
