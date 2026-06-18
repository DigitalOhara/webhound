package session

import (
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

// TargetState holds per-target scan progress.
type TargetState struct {
	CompletedPaths map[string]bool     `json:"completed_paths"`
	Results        []*response.Result  `json:"results"`
	RecursionDecisions map[string]bool `json:"recursion_decisions"` // path -> recurse
	StartedAt      time.Time           `json:"started_at"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
}

// State is the full serialisable scan state written to disk on checkpoint.
type State struct {
	ID              string                     `json:"id"`
	Config          *config.ScanConfig         `json:"config"`
	Targets         map[string]*TargetState    `json:"targets"`
	StartedAt       time.Time                  `json:"started_at"`
	CheckpointedAt  time.Time                  `json:"checkpointed_at"`
	CompletedAt     *time.Time                 `json:"completed_at,omitempty"`
	TotalRequests   int64                      `json:"total_requests"`
}

// NewState initialises an empty State.
func NewState(id string, cfg *config.ScanConfig, targets []string) *State {
	s := &State{
		ID:        id,
		Config:    cfg,
		Targets:   make(map[string]*TargetState, len(targets)),
		StartedAt: time.Now(),
	}
	for _, t := range targets {
		s.Targets[t] = &TargetState{
			CompletedPaths:     make(map[string]bool),
			RecursionDecisions: make(map[string]bool),
			StartedAt:          time.Now(),
		}
	}
	return s
}

// MarkPathComplete records that a path has been scanned for the given target.
func (s *State) MarkPathComplete(target, path string) {
	if ts, ok := s.Targets[target]; ok {
		ts.CompletedPaths[path] = true
	}
}

// IsPathComplete returns true if the path has already been scanned.
func (s *State) IsPathComplete(target, path string) bool {
	if ts, ok := s.Targets[target]; ok {
		return ts.CompletedPaths[path]
	}
	return false
}

// AddResult appends a result for a target.
func (s *State) AddResult(target string, r *response.Result) {
	if ts, ok := s.Targets[target]; ok {
		ts.Results = append(ts.Results, r)
	}
	s.TotalRequests++
}

// SetRecursionDecision records whether a directory should be recursed.
func (s *State) SetRecursionDecision(target, path string, recurse bool) {
	if ts, ok := s.Targets[target]; ok {
		ts.RecursionDecisions[path] = recurse
	}
}

// GetRecursionDecision retrieves a stored recursion decision.
// The bool return indicates whether a decision was recorded.
func (s *State) GetRecursionDecision(target, path string) (recurse bool, found bool) {
	if ts, ok := s.Targets[target]; ok {
		if decision, ok := ts.RecursionDecisions[path]; ok {
			return decision, true
		}
	}
	return false, false
}

// AllResults returns all results across all targets in order.
func (s *State) AllResults() []*response.Result {
	var all []*response.Result
	for _, ts := range s.Targets {
		all = append(all, ts.Results...)
	}
	return all
}
