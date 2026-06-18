package wordlist

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/pkg/utils"
)

// Manager loads, deduplicates, and caches wordlists.
type Manager struct {
	mu      sync.Once
	entries []string
	cfg     *config.ScanConfig
}

// NewManager creates a Manager.
func NewManager(cfg *config.ScanConfig) *Manager {
	return &Manager{cfg: cfg}
}

// Load initialises the wordlist (idempotent — subsequent calls are no-ops).
func (m *Manager) Load() error {
	var loadErr error
	m.mu.Do(func() {
		var raw []string

		if len(m.cfg.Wordlists) == 0 {
			// Use the built-in common wordlist.
			raw = BuiltinCommon()
		} else {
			for _, path := range m.cfg.Wordlists {
				switch path {
				case "common":
					raw = append(raw, BuiltinCommon()...)
				case "directories":
					raw = append(raw, BuiltinDirectories()...)
				case "files":
					raw = append(raw, BuiltinFiles()...)
				default:
					lines, err := loadFile(path)
					if err != nil {
						loadErr = fmt.Errorf("loading wordlist %q: %w", path, err)
						return
					}
					raw = append(raw, lines...)
				}
			}
		}

		m.entries = utils.DeduplicateStrings(raw)
	})
	return loadErr
}

// Entries returns the deduplicated base wordlist (without extension expansion).
func (m *Manager) Entries() []string {
	return m.entries
}

// Count returns the number of base wordlist entries.
func (m *Manager) Count() int {
	return len(m.entries)
}

func loadFile(path string) ([]string, error) {
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
