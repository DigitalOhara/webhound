package response

import "github.com/digitalohara/webhound/internal/config"

// Filter holds compiled filter predicates derived from config.
type Filter struct {
	allowStatus    map[int]bool
	denyStatus     map[int]bool
	minLength      int64
	maxLength      int64
	hideWords      map[int]bool
	hideLines      map[int]bool
	hideLength     map[int64]bool
}

// NewFilter builds a Filter from ScanConfig.
func NewFilter(cfg *config.ScanConfig) *Filter {
	f := &Filter{
		allowStatus: make(map[int]bool),
		denyStatus:  make(map[int]bool),
		hideWords:   make(map[int]bool),
		hideLines:   make(map[int]bool),
		hideLength:  make(map[int64]bool),
		minLength:   int64(cfg.MinLength),
		maxLength:   int64(cfg.MaxLength),
	}
	for _, s := range cfg.StatusCodes {
		f.allowStatus[s] = true
	}
	for _, s := range cfg.StatusCodesBlacklist {
		f.denyStatus[s] = true
	}
	for _, w := range cfg.HideWords {
		f.hideWords[w] = true
	}
	for _, l := range cfg.HideLines {
		f.hideLines[l] = true
	}
	for _, l := range cfg.HideLength {
		f.hideLength[int64(l)] = true
	}
	return f
}

// Pass returns true if the result should be reported.
func (f *Filter) Pass(r *Result) bool {
	// Status code allowlist (if set)
	if len(f.allowStatus) > 0 && !f.allowStatus[r.StatusCode] {
		return false
	}
	// Status code denylist
	if f.denyStatus[r.StatusCode] {
		return false
	}
	// Content-length bounds
	if f.minLength > 0 && r.ContentLength < f.minLength {
		return false
	}
	if f.maxLength > 0 && r.ContentLength > f.maxLength {
		return false
	}
	// Hidden lengths / words / lines
	if f.hideLength[r.ContentLength] {
		return false
	}
	if f.hideWords[r.Words] {
		return false
	}
	if f.hideLines[r.Lines] {
		return false
	}
	return true
}
