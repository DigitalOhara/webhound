package response

import (
	"net/http"
	"strings"
	"time"

	"github.com/digitalohara/webhound/pkg/utils"
)

// Job is a single unit of work sent to a worker.
type Job struct {
	ID      string
	BaseURL string
	Path    string
	Method  string
	Depth   int
}

// RawResult is the unprocessed response from the HTTP layer.
type RawResult struct {
	Job          Job
	StatusCode   int
	Headers      http.Header
	Body         []byte
	ContentLength int64
	ResponseTime  time.Duration
	Error        error
	FinalURL     string // URL after any redirects
	IsRedirect   bool
}

// Result is a processed, filter-passed finding.
type Result struct {
	URL           string
	Path          string
	StatusCode    int
	ContentLength int64
	ContentType   string
	Words         int
	Lines         int
	ResponseTime  time.Duration
	RedirectURL   string
	Depth         int
	IsDirectory   bool
	FoundAt       time.Time
	Target        string
}

// Analyzer converts RawResults into Results applying wildcard suppression.
type Analyzer struct {
	wildcardDetector *WildcardDetector
	filter           *Filter
}

// NewAnalyzer creates an Analyzer.
func NewAnalyzer(wd *WildcardDetector, f *Filter) *Analyzer {
	return &Analyzer{wildcardDetector: wd, filter: f}
}

// Analyze converts a RawResult to a Result.
// Returns nil when the result should be suppressed.
func (a *Analyzer) Analyze(raw *RawResult) *Result {
	if raw.Error != nil {
		return nil
	}
	if raw.StatusCode == 0 {
		return nil
	}

	body := string(raw.Body)
	ct := raw.Headers.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}

	cl := raw.ContentLength
	if cl <= 0 && len(raw.Body) > 0 {
		cl = int64(len(raw.Body))
	}

	r := &Result{
		URL:           raw.FinalURL,
		Path:          raw.Job.Path,
		StatusCode:    raw.StatusCode,
		ContentLength: cl,
		ContentType:   ct,
		Words:         utils.CountWords(body),
		Lines:         utils.CountLines(body),
		ResponseTime:  raw.ResponseTime,
		Depth:         raw.Job.Depth,
		FoundAt:       time.Now(),
		Target:        raw.Job.BaseURL,
	}

	if raw.IsRedirect {
		r.RedirectURL = raw.FinalURL
		r.URL = raw.Job.BaseURL + "/" + strings.TrimLeft(raw.Job.Path, "/")
	}

	r.IsDirectory = looksLikeDirectory(raw.Job.Path, raw.StatusCode, ct)

	// Wildcard suppression
	if a.wildcardDetector != nil {
		if a.wildcardDetector.IsFalsePositive(r, raw.Body) {
			return nil
		}
	}

	// Filter predicates
	if a.filter != nil && !a.filter.Pass(r) {
		return nil
	}

	return r
}

// looksLikeDirectory heuristically determines if a path is a directory.
func looksLikeDirectory(path string, status int, ct string) bool {
	if strings.HasSuffix(path, "/") {
		return true
	}
	if strings.Contains(path, ".") {
		return false
	}
	if status == http.StatusMovedPermanently || status == http.StatusFound {
		return true
	}
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/json") {
		if !strings.Contains(path, ".") {
			return true
		}
	}
	return false
}
