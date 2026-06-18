package recursion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

// SmartPriorityPaths are directory names that get preferential recursion.
var SmartPriorityPaths = []string{
	"admin", "api", "portal", "dashboard", "uploads", "internal",
	"backup", "config", "secret", "dev", "staging", "manage",
	"management", "console", "control", "panel", "secure",
}

// Manager orchestrates all recursion modes.
type Manager struct {
	cfg         *config.ScanConfig
	interactive *InteractiveMode
	queue       *QueueManager
	wordlistLen int
	rps         float64
}

// NewManager creates a recursion Manager.
func NewManager(cfg *config.ScanConfig, wordlistLen int) *Manager {
	m := &Manager{
		cfg:         cfg,
		wordlistLen: wordlistLen,
		rps:         cfg.Rate,
	}
	if cfg.InteractiveRecursion {
		m.interactive = NewInteractiveMode()
	}
	if cfg.QueueRecursion {
		m.queue = NewQueueManager()
	}
	return m
}

// ShouldRecurse decides whether to recurse into a discovered directory.
// For interactive mode it blocks for user input.
// For queue mode it enqueues and returns false (deferred decision).
// For auto mode it returns true if within depth limit.
func (m *Manager) ShouldRecurse(ctx context.Context, r *response.Result) bool {
	if !m.cfg.Recursive && !m.cfg.InteractiveRecursion && !m.cfg.QueueRecursion {
		return false
	}
	if !r.IsDirectory {
		return false
	}
	if r.Depth >= m.cfg.MaxDepth {
		return false
	}
	if m.isExcluded(r.Path) {
		return false
	}

	switch {
	case m.cfg.QueueRecursion:
		m.queue.Enqueue(r)
		return false

	case m.cfg.InteractiveRecursion:
		est := m.estimatedRequests()
		dur := m.estimatedDuration(est)
		return m.interactive.Prompt(r, est, dur)

	case m.cfg.Recursive:
		if m.cfg.SmartRecursion {
			return m.isHighPriority(r.Path) || isGoodStatus(r.StatusCode)
		}
		return isGoodStatus(r.StatusCode)

	default:
		return false
	}
}

// FlushQueue presents the deferred queue to the user and returns selected dirs.
// Only meaningful when QueueRecursion is enabled.
func (m *Manager) FlushQueue(ctx context.Context) ([]*response.Result, error) {
	if m.queue == nil || !m.queue.HasItems() {
		return nil, nil
	}
	selected, err := m.queue.ReviewAndSelect()
	if err != nil {
		return nil, fmt.Errorf("queue review: %w", err)
	}
	return selected, nil
}

// ConfirmRecurse prints a summary and asks for explicit user confirmation
// before starting a large recursion. Returns false if the user cancels.
func (m *Manager) ConfirmRecurse(r *response.Result) bool {
	if m.cfg.NoConfirm {
		return true
	}
	est := m.estimatedRequests()
	dur := m.estimatedDuration(est)
	fmt.Printf("\n\033[1;34m[*]\033[0m Preparing recursion into: %s\n", r.URL)
	fmt.Printf("    Estimated requests : %d\n", est)
	fmt.Printf("    Estimated duration : %s\n", dur)
	fmt.Printf("    Current rate       : %.0f req/s\n\n", m.rps)
	fmt.Printf("    Proceed? (Y/N): ")

	var input string
	fmt.Scanln(&input)
	return strings.ToUpper(strings.TrimSpace(input)) == "Y"
}

func (m *Manager) estimatedRequests() int {
	return m.wordlistLen
}

func (m *Manager) estimatedDuration(requests int) string {
	if m.rps <= 0 {
		return "unknown"
	}
	secs := float64(requests) / m.rps
	return (time.Duration(secs * float64(time.Second))).Round(time.Second).String()
}

func (m *Manager) isExcluded(path string) bool {
	for _, ex := range m.cfg.ExcludeRecursion {
		if strings.Contains(path, ex) {
			return true
		}
	}
	return false
}

func (m *Manager) isHighPriority(path string) bool {
	lower := strings.ToLower(path)
	for _, p := range SmartPriorityPaths {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isGoodStatus(code int) bool {
	return code == 200 || code == 301 || code == 302 || code == 307
}
