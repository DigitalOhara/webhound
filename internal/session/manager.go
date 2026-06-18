package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles session persistence: save, checkpoint, resume.
type Manager struct {
	mu                 sync.Mutex
	state              *State
	path               string
	checkpointInterval int
	requestsSinceCP    int
}

// NewManager creates a session Manager backed by path.
func NewManager(state *State, path string, checkpointInterval int) *Manager {
	if checkpointInterval <= 0 {
		checkpointInterval = 500
	}
	return &Manager{
		state:              state,
		path:               path,
		checkpointInterval: checkpointInterval,
	}
}

// State returns the underlying State.
func (m *Manager) State() *State { return m.state }

// RecordRequest increments the request counter and checkpoints if needed.
func (m *Manager) RecordRequest(target, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.MarkPathComplete(target, path)
	m.requestsSinceCP++
	if m.requestsSinceCP >= m.checkpointInterval {
		m.requestsSinceCP = 0
		return m.writeUnlocked()
	}
	return nil
}

// Save persists the current state immediately.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeUnlocked()
}

// writeUnlocked serialises state to disk atomically (write-then-rename).
func (m *Manager) writeUnlocked() error {
	m.state.CheckpointedAt = time.Now()
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling session: %w", err)
	}
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing session temp file: %w", err)
	}
	return os.Rename(tmp, m.path)
}

// Load reads a session file from disk.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	return &s, nil
}

// SessionDir returns the directory used for session files.
func SessionDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "webhound", "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// SessionPath returns the full path for a session by ID.
func SessionPath(id string) (string, error) {
	dir, err := SessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// ListSessions returns all stored session IDs.
func ListSessions() ([]string, error) {
	dir, err := SessionDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			ids = append(ids, e.Name()[:len(e.Name())-5])
		}
	}
	return ids, nil
}
