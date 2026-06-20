package engine

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digitalohara/webhound/internal/auth"
	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/input"
	"github.com/digitalohara/webhound/internal/ratelimit"
	"github.com/digitalohara/webhound/internal/recursion"
	"github.com/digitalohara/webhound/internal/reporting"
	"github.com/digitalohara/webhound/internal/response"
	"github.com/digitalohara/webhound/internal/session"
	"github.com/digitalohara/webhound/internal/wordlist"
	"github.com/digitalohara/webhound/pkg/logger"
	"github.com/digitalohara/webhound/pkg/utils"
)

const version = "1.0.1"

// Orchestrator wires all subsystems and runs the scan loop.
type Orchestrator struct {
	cfg          *config.ScanConfig
	authMgr      *auth.Manager
	wordlistMgr  *wordlist.Manager
	generator    *wordlist.Generator
	rateMgr      *ratelimit.Manager
	sessionMgr   *session.Manager
	console      *reporting.ConsoleReporter
	pool         *WorkerPool
	requester    *Requester
	wildcard     *response.WildcardDetector
	analyzer     *response.Analyzer
	filter       *response.Filter
	httpClient   *http.Client
	startedAt    time.Time
	allResults   []*response.Result
}

// New constructs an Orchestrator from cfg.
func New(cfg *config.ScanConfig) (*Orchestrator, error) {
	o := &Orchestrator{cfg: cfg, startedAt: time.Now()}

	// HTTP transport + client
	tr, err := BuildTransport(transportConfig{
		ProxyURL:           cfg.Proxy,
		CACert:             cfg.CACert,
		TLSMinVersion:      cfg.TLSMinVersion,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MaxConnsPerHost:    cfg.Threads + 5,
		Timeout:            cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("building transport: %w", err)
	}
	o.httpClient = &http.Client{
		Transport: tr,
		Timeout:   cfg.Timeout + 5*time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !cfg.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= cfg.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Auth
	authMgr, err := auth.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("setting up auth: %w", err)
	}
	o.authMgr = authMgr

	// Rate limiter
	o.rateMgr = ratelimit.NewManager(cfg.Rate, cfg.Delay)

	// Wordlist
	o.wordlistMgr = wordlist.NewManager(cfg)
	if err := o.wordlistMgr.Load(); err != nil {
		return nil, fmt.Errorf("loading wordlist: %w", err)
	}
	o.generator = wordlist.NewGenerator(o.wordlistMgr, cfg.Extensions, cfg.NoExtension)

	// Requester + pool
	o.requester = NewRequester(o.httpClient, o.authMgr, o.rateMgr, cfg.UserAgent, cfg.MaxRetries, cfg.Method)
	o.pool = NewWorkerPool(cfg.Threads, o.requester)

	// Response analysis
	o.wildcard = response.NewWildcardDetector(o.httpClient, cfg.UserAgent)
	o.filter = response.NewFilter(cfg)
	o.analyzer = response.NewAnalyzer(o.wildcard, o.filter)

	// Console reporter
	o.console = reporting.NewConsoleReporter(cfg.Quiet, cfg.NoColor)

	return o, nil
}

// Run executes the full scan lifecycle.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Handle OS signals for graceful shutdown.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\n\n\033[1;33m[!]\033[0m Interrupt received — finishing current level and saving session...")
			cancel()
		case <-ctx.Done():
		}
	}()

	// Load targets
	inputMgr := input.NewManager(o.cfg)
	targets, err := inputMgr.LoadTargets()
	if err != nil {
		return err
	}

	// Session setup
	sessionID := o.cfg.Session
	if sessionID == "" {
		sessionID = "scan-" + time.Now().Format("2006-01-02-150405")
	}

	var state *session.State
	if o.cfg.Resume != "" {
		path, err := session.SessionPath(o.cfg.Resume)
		if err != nil {
			return fmt.Errorf("resolving session path: %w", err)
		}
		state, err = session.Load(path)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\033[1;34m[*]\033[0m Resuming session %q (%d targets)\n", o.cfg.Resume, len(targets))
	} else {
		state = session.NewState(sessionID, o.cfg, targets)
	}

	sessionPath, err := session.SessionPath(sessionID)
	if err != nil {
		logger.Warn().Err(err).Msg("could not resolve session path; checkpointing disabled")
	}
	sessionMgr := session.NewManager(state, sessionPath, o.cfg.CheckpointInterval)
	o.sessionMgr = sessionMgr

	// Expanded paths (base wordlist × extensions)
	expandedPaths := o.generator.ExpandedPaths()
	totalJobs := int64(len(expandedPaths)) * int64(len(targets))
	o.console.SetTotalJobs(totalJobs)

	// Print banner for first target
	if len(targets) > 0 {
		o.console.PrintBanner(version, targets[0], o.cfg.Threads, o.cfg.Rate)
	}

	// Progress ticker
	if !o.cfg.Quiet {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					o.console.PrintProgress()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Scan each target
	for _, target := range targets {
		if ctx.Err() != nil {
			break
		}
		o.console.SetTarget(target)
		if err := o.scanTarget(ctx, target, expandedPaths, state); err != nil {
			logger.Error().Err(err).Str("target", target).Msg("target scan error")
		}
	}

	// Save final session
	if sessionMgr != nil {
		_ = sessionMgr.Save()
	}

	// Print summary
	o.console.PrintSummary(len(targets))

	// Write reports
	if err := o.writeReports(targets); err != nil {
		logger.Error().Err(err).Msg("writing reports")
	}

	return nil
}

// scanTarget executes a BFS-style scan of a single target URL.
func (o *Orchestrator) scanTarget(ctx context.Context, target string, expandedPaths []string, state *session.State) error {
	// Wildcard probe
	if _, err := o.wildcard.Probe(ctx, target); err != nil {
		logger.Warn().Err(err).Str("target", target).Msg("wildcard probe failed")
	}

	type level struct {
		baseURL string
		depth   int
	}

	recursionMgr := recursion.NewManager(o.cfg, len(expandedPaths))
	queue := []level{{baseURL: target, depth: 0}}

	for len(queue) > 0 && ctx.Err() == nil {
		current := queue[0]
		queue = queue[1:]

		if current.depth > o.cfg.MaxDepth {
			continue
		}

		logger.Debug().Str("base", current.baseURL).Int("depth", current.depth).Msg("scanning level")

		// Build jobs for this level, skipping already-completed paths.
		var jobs []response.Job
		for _, path := range expandedPaths {
			fullPath := utils.JoinURL(current.baseURL, path)
			if state.IsPathComplete(target, fullPath) {
				o.console.IncrementCompleted()
				continue
			}
			jobs = append(jobs, response.Job{
				BaseURL: current.baseURL,
				Path:    path,
				Method:  o.cfg.Method,
				Depth:   current.depth,
			})
		}

		// Stream results from workers as they complete (real-time progress).
		jobCh := make(chan response.Job, o.cfg.Threads*2)
		resultCh := make(chan *response.RawResult, o.cfg.Threads*2)
		o.pool.Run(ctx, jobCh, resultCh)

		go func() {
			defer close(jobCh)
			for _, job := range jobs {
				select {
				case jobCh <- job:
				case <-ctx.Done():
					return
				}
			}
		}()

		for raw := range resultCh {
			// Checkpoint
			if o.sessionMgr != nil {
				if err := o.sessionMgr.RecordRequest(target, utils.JoinURL(raw.Job.BaseURL, raw.Job.Path)); err != nil {
					logger.Debug().Err(err).Msg("checkpoint error")
				}
			}

			result := o.analyzer.Analyze(raw)
			if result == nil {
				o.console.IncrementCompleted()
				continue
			}

			// Store and report.
			o.allResults = append(o.allResults, result)
			state.AddResult(target, result)
			o.console.Report(result)

			// Consider recursion.
			if result.IsDirectory && recursionMgr.ShouldRecurse(ctx, result) {
				if !o.cfg.NoConfirm && o.cfg.Recursive {
					if !recursionMgr.ConfirmRecurse(result) {
						continue
					}
				}
				queue = append(queue, level{
					baseURL: result.URL,
					depth:   current.depth + 1,
				})
			}
		}

		// Flush queue-mode deferred recursion at end of each level.
		if o.cfg.QueueRecursion {
			selected, err := recursionMgr.FlushQueue(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("queue flush")
			}
			for _, r := range selected {
				queue = append(queue, level{
					baseURL: r.URL,
					depth:   r.Depth + 1,
				})
			}
		}
	}

	return nil
}

// writeReports generates output files based on cfg.Format / cfg.Output.
func (o *Orchestrator) writeReports(targets []string) error {
	if o.cfg.Output == "" {
		return nil
	}

	switch o.cfg.Format {
	case "json", "":
		if err := reporting.WriteJSON(o.cfg.Output, o.cfg, o.allResults, targets, o.startedAt); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\033[1;34m[*]\033[0m JSON report saved: %s\n", o.cfg.Output)
	case "csv":
		if err := reporting.WriteCSV(o.cfg.Output, o.allResults); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\033[1;34m[*]\033[0m CSV report saved: %s\n", o.cfg.Output)
	case "html":
		if err := reporting.WriteHTML(o.cfg.Output, o.cfg, o.allResults, targets, o.startedAt); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\033[1;34m[*]\033[0m HTML report saved: %s\n", o.cfg.Output)
	case "txt":
		if err := reporting.WriteTXT(o.cfg.Output, o.cfg, o.allResults, targets, o.startedAt); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\033[1;34m[*]\033[0m Results saved: %s\n", o.cfg.Output)
	default:
		return fmt.Errorf("unknown output format %q", o.cfg.Format)
	}
	return nil
}
