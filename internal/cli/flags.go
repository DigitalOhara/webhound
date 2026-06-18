package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/digitalohara/webhound/internal/config"
)

// bindScanFlags attaches all scan flags to cmd and returns a pointer
// to the ScanConfig that will be populated when the command runs.
func bindScanFlags(cmd *cobra.Command) *config.ScanConfig {
	cfg := config.DefaultConfig()

	// Target
	cmd.Flags().StringVarP(&cfg.URL, "url", "u", "", "Target URL (single target)")
	cmd.Flags().StringVarP(&cfg.File, "file", "f", "", "File containing target URLs (one per line)")

	// Auth
	cmd.Flags().StringVar(&cfg.BearerToken, "bearer-token", "", "Bearer token for Authorization header")
	cmd.Flags().StringArrayVarP(&cfg.Headers, "header", "H", nil, "Custom header (repeatable: -H 'X-Foo: bar')")
	cmd.Flags().StringArrayVar(&cfg.Cookies, "cookie", nil, "Cookie string (repeatable: --cookie 'key=val')")
	cmd.Flags().StringVar(&cfg.CookieFile, "cookie-file", "", "Load cookies from file (Netscape or key=value format)")
	cmd.Flags().StringVar(&cfg.BasicAuth, "basic-auth", "", "HTTP Basic auth credentials (user:pass)")
	cmd.Flags().StringVar(&cfg.Profile, "profile", "", "Auth profile name to load")

	// Wordlists
	cmd.Flags().StringArrayVarP(&cfg.Wordlists, "wordlist", "w", nil, "Wordlist file(s) or built-in name: common, directories, files")
	cmd.Flags().StringSliceVarP(&cfg.Extensions, "extensions", "x", cfg.Extensions, "File extensions to probe")
	cmd.Flags().BoolVar(&cfg.NoExtension, "no-extension", false, "Do not append extensions to wordlist entries")

	// HTTP tuning
	cmd.Flags().IntVarP(&cfg.Threads, "threads", "t", cfg.Threads, fmt.Sprintf("Concurrent workers (max %d)", config.MaxThreads))
	cmd.Flags().Float64VarP(&cfg.Rate, "rate", "r", cfg.Rate, "Maximum requests per second (0 = unlimited)")
	cmd.Flags().DurationVar(&cfg.Delay, "delay", 0, "Fixed delay between requests per worker (e.g. 200ms)")
	cmd.Flags().DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "HTTP request timeout")
	cmd.Flags().IntVar(&cfg.MaxRetries, "max-retries", cfg.MaxRetries, "Maximum retries per request")
	cmd.Flags().StringVar(&cfg.UserAgent, "user-agent", cfg.UserAgent, "HTTP User-Agent string")
	cmd.Flags().BoolVar(&cfg.FollowRedirects, "follow-redirects", cfg.FollowRedirects, "Follow HTTP redirects")
	cmd.Flags().IntVar(&cfg.MaxRedirects, "max-redirects", cfg.MaxRedirects, "Maximum redirect hops")
	cmd.Flags().StringVar(&cfg.Method, "method", cfg.Method, "HTTP method (GET, POST, HEAD, …)")

	// Proxy / TLS
	cmd.Flags().StringVarP(&cfg.Proxy, "proxy", "p", "", "Proxy URL (http://, https://, socks5://)")
	cmd.Flags().BoolVar(&cfg.InsecureSkipVerify, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().StringVar(&cfg.CACert, "ca-cert", "", "Custom CA certificate file")
	cmd.Flags().StringVar(&cfg.TLSMinVersion, "tls-version", "1.2", "Minimum TLS version (1.0, 1.1, 1.2)")

	// Filtering
	cmd.Flags().IntSliceVar(&cfg.StatusCodes, "status-codes", cfg.StatusCodes, "Status codes to include")
	cmd.Flags().IntSliceVar(&cfg.StatusCodesBlacklist, "status-codes-blacklist", nil, "Status codes to exclude")
	cmd.Flags().IntVar(&cfg.MinLength, "min-length", 0, "Minimum content length to include")
	cmd.Flags().IntVar(&cfg.MaxLength, "max-length", 0, "Maximum content length to include (0 = no limit)")
	cmd.Flags().IntSliceVar(&cfg.HideLength, "hide-length", nil, "Suppress results with these exact content lengths")
	cmd.Flags().IntSliceVar(&cfg.HideWords, "hide-words", nil, "Suppress results with these word counts")
	cmd.Flags().IntSliceVar(&cfg.HideLines, "hide-lines", nil, "Suppress results with these line counts")

	// Recursion
	cmd.Flags().BoolVar(&cfg.Recursive, "recursive", false, "Automatically recurse into discovered directories")
	cmd.Flags().BoolVar(&cfg.InteractiveRecursion, "interactive-recursion", false, "Prompt before recursing into each directory")
	cmd.Flags().BoolVar(&cfg.QueueRecursion, "queue-recursion", false, "Collect directories and select at end of scan")
	cmd.Flags().IntVar(&cfg.MaxDepth, "max-depth", cfg.MaxDepth, "Maximum recursion depth")
	cmd.Flags().StringSliceVar(&cfg.ExcludeRecursion, "exclude-recursion", nil, "Path patterns to skip for recursion")
	cmd.Flags().BoolVar(&cfg.SmartRecursion, "smart-recursion", cfg.SmartRecursion, "Prioritise high-value directories for recursion")

	// Output
	cmd.Flags().StringVarP(&cfg.Output, "output", "o", "", "Output file path")
	cmd.Flags().StringVar(&cfg.Format, "format", "", "Output format: json, csv, html (inferred from --output extension if omitted)")
	cmd.Flags().BoolVarP(&cfg.Quiet, "quiet", "q", false, "Suppress all output except findings")
	cmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable debug logging")
	cmd.Flags().BoolVar(&cfg.NoColor, "no-color", false, "Disable colour output")

	// Session
	cmd.Flags().StringVar(&cfg.Session, "session", "", "Session name (default: auto-generated timestamp)")
	cmd.Flags().StringVar(&cfg.Resume, "resume", "", "Session ID to resume")
	cmd.Flags().IntVar(&cfg.CheckpointInterval, "checkpoint-interval", cfg.CheckpointInterval, "Checkpoint every N requests")

	// Safety
	cmd.Flags().BoolVar(&cfg.NoConfirm, "no-confirm", false, "Skip recursion confirmation prompts")
	cmd.Flags().IntVar(&cfg.MaxRequests, "max-requests", 0, "Hard cap on total requests (0 = no limit)")

	return cfg
}

// inferFormat deduces the output format from the file extension when --format
// was not explicitly supplied.
func inferFormat(cfg *config.ScanConfig) {
	if cfg.Format != "" || cfg.Output == "" {
		return
	}
	lower := strings.ToLower(cfg.Output)
	switch {
	case strings.HasSuffix(lower, ".json"):
		cfg.Format = "json"
	case strings.HasSuffix(lower, ".csv"):
		cfg.Format = "csv"
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		cfg.Format = "html"
	default:
		cfg.Format = "json"
	}
}

// loadConfigFile merges a YAML config file into cfg if --config was specified.
func loadConfigFile(cfgFile string, cfg *config.ScanConfig) error {
	if cfgFile == "" {
		return nil
	}
	fileCfg, err := config.LoadFile(cfgFile)
	if err != nil {
		return err
	}
	// Only apply file values for fields the user didn't set via flags.
	// We merge by checking zero-values; explicit flags always win.
	if cfg.URL == "" {
		cfg.URL = fileCfg.URL
	}
	if cfg.File == "" {
		cfg.File = fileCfg.File
	}
	if cfg.BearerToken == "" {
		cfg.BearerToken = fileCfg.BearerToken
	}
	if len(cfg.Wordlists) == 0 {
		cfg.Wordlists = fileCfg.Wordlists
	}
	return nil
}

// printConfigFile writes an example default.yaml to stdout.
func printConfigFile() {
	cfg := config.DefaultConfig()
	fmt.Fprintf(os.Stdout, "# WebHound default configuration\n")
	fmt.Fprintf(os.Stdout, "threads: %d\n", cfg.Threads)
	fmt.Fprintf(os.Stdout, "rate: %.0f\n", cfg.Rate)
	fmt.Fprintf(os.Stdout, "timeout: %s\n", cfg.Timeout)
	fmt.Fprintf(os.Stdout, "max_retries: %d\n", cfg.MaxRetries)
	fmt.Fprintf(os.Stdout, "max_depth: %d\n", cfg.MaxDepth)
	fmt.Fprintf(os.Stdout, "follow_redirects: %v\n", cfg.FollowRedirects)
	fmt.Fprintf(os.Stdout, "user_agent: %q\n", cfg.UserAgent)
	fmt.Fprintf(os.Stdout, "recursive: false\n")
	fmt.Fprintf(os.Stdout, "smart_recursion: true\n")
	fmt.Fprintf(os.Stdout, "no_confirm: false\n")
	_ = time.Second // suppress unused import warning
}
