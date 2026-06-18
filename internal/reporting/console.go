package reporting

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/digitalohara/webhound/internal/response"
)

// statusColor maps HTTP status ranges to terminal colours.
func statusColor(code int) *color.Color {
	switch {
	case code >= 500:
		return color.New(color.FgRed, color.Bold)
	case code >= 400:
		return color.New(color.FgYellow)
	case code >= 300:
		return color.New(color.FgCyan)
	case code >= 200:
		return color.New(color.FgGreen, color.Bold)
	default:
		return color.New(color.FgWhite)
	}
}

// ConsoleReporter prints real-time findings and progress to stdout/stderr.
type ConsoleReporter struct {
	mu            sync.Mutex
	quiet         bool
	noColor       bool
	startTime     time.Time
	totalJobs     int64
	completedJobs atomic.Int64
	foundCount    atomic.Int64
	currentTarget string
	lastStatus    string
}

// NewConsoleReporter creates a ConsoleReporter.
func NewConsoleReporter(quiet, noColor bool) *ConsoleReporter {
	if noColor {
		color.NoColor = true
	}
	return &ConsoleReporter{
		quiet:     quiet,
		noColor:   noColor,
		startTime: time.Now(),
	}
}

// SetTotalJobs sets the total job count for progress calculation.
func (c *ConsoleReporter) SetTotalJobs(n int64) {
	atomic.StoreInt64(&c.totalJobs, n)
}

// SetTarget updates the current target label shown in the status line.
func (c *ConsoleReporter) SetTarget(target string) {
	c.mu.Lock()
	c.currentTarget = target
	c.mu.Unlock()
}

// PrintBanner prints the startup banner.
func (c *ConsoleReporter) PrintBanner(version, target string, threads int, rate float64) {
	if c.quiet {
		return
	}
	bold := color.New(color.FgCyan, color.Bold)
	fmt.Fprintln(os.Stderr)
	bold.Fprintln(os.Stderr, "  ██╗    ██╗███████╗██████╗ ██╗  ██╗ ██████╗ ██╗   ██╗███╗   ██╗██████╗ ")
	bold.Fprintln(os.Stderr, "  ██║    ██║██╔════╝██╔══██╗██║  ██║██╔═══██╗██║   ██║████╗  ██║██╔══██╗")
	bold.Fprintln(os.Stderr, "  ██║ █╗ ██║█████╗  ██████╔╝███████║██║   ██║██║   ██║██╔██╗ ██║██║  ██║")
	bold.Fprintln(os.Stderr, "  ██║███╗██║██╔══╝  ██╔══██╗██╔══██║██║   ██║██║   ██║██║╚██╗██║██║  ██║")
	bold.Fprintln(os.Stderr, "  ╚███╔███╔╝███████╗██████╔╝██║  ██║╚██████╔╝╚██████╔╝██║ ╚████║██████╔╝")
	bold.Fprintln(os.Stderr, "   ╚══╝╚══╝ ╚══════╝╚═════╝ ╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚═════╝")
	fmt.Fprintf(os.Stderr, "  Web Content Discovery Tool v%s\n\n", version)
	fmt.Fprintf(os.Stderr, "  Target  : %s\n", target)
	fmt.Fprintf(os.Stderr, "  Threads : %d\n", threads)
	fmt.Fprintf(os.Stderr, "  Rate    : %.0f req/s\n", rate)
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 72))
	fmt.Fprintln(os.Stderr)
}

// Report prints a single result to stdout.
func (c *ConsoleReporter) Report(r *response.Result) {
	c.completedJobs.Add(1)
	c.foundCount.Add(1)

	code := fmt.Sprintf("%d", r.StatusCode)
	coloredCode := statusColor(r.StatusCode).Sprint(code)

	size := formatSize(r.ContentLength)
	redir := ""
	if r.RedirectURL != "" {
		redir = fmt.Sprintf(" -> %s", r.RedirectURL)
	}
	dirMark := ""
	if r.IsDirectory {
		dirMark = color.New(color.FgMagenta).Sprint(" [DIR]")
	}

	line := fmt.Sprintf("[%s] %-72s [%s] [%s]%s%s",
		coloredCode,
		r.URL,
		size,
		r.ResponseTime.Round(time.Millisecond),
		redir,
		dirMark,
	)
	fmt.Println(line)
}

// IncrementCompleted records a job completion without a finding (for progress tracking).
func (c *ConsoleReporter) IncrementCompleted() {
	c.completedJobs.Add(1)
}

// PrintProgress writes the current progress to stderr (overwrites previous line).
func (c *ConsoleReporter) PrintProgress() {
	if c.quiet {
		return
	}
	completed := c.completedJobs.Load()
	total := atomic.LoadInt64(&c.totalJobs)
	found := c.foundCount.Load()
	elapsed := time.Since(c.startTime)

	var pct float64
	if total > 0 {
		pct = float64(completed) / float64(total) * 100
	}
	var rps float64
	if elapsed.Seconds() > 0 {
		rps = float64(completed) / elapsed.Seconds()
	}

	c.mu.Lock()
	target := c.currentTarget
	c.mu.Unlock()

	status := fmt.Sprintf("\r\033[K  Progress: %d/%d (%.1f%%) | Found: %d | %.0f req/s | %s | %s     ",
		completed, total, pct, found, rps, elapsed.Round(time.Second), target)

	if status != c.lastStatus {
		fmt.Fprint(os.Stderr, status)
		c.lastStatus = status
	}
}

// PrintSummary prints final scan statistics.
func (c *ConsoleReporter) PrintSummary(targets int) {
	if c.quiet {
		return
	}
	elapsed := time.Since(c.startTime)
	found := c.foundCount.Load()
	completed := c.completedJobs.Load()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  \033[1;32m[✓]\033[0m Scan complete\n")
	fmt.Fprintf(os.Stderr, "      Targets    : %d\n", targets)
	fmt.Fprintf(os.Stderr, "      Requests   : %d\n", completed)
	fmt.Fprintf(os.Stderr, "      Found      : %d\n", found)
	fmt.Fprintf(os.Stderr, "      Duration   : %s\n", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		fmt.Fprintf(os.Stderr, "      Rate       : %.0f req/s\n", float64(completed)/elapsed.Seconds())
	}
	fmt.Fprintln(os.Stderr)
}

func formatSize(n int64) string {
	if n < 0 {
		return "-"
	}
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
