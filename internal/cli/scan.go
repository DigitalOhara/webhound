package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/digitalohara/webhound/internal/engine"
	"github.com/digitalohara/webhound/pkg/logger"
	"github.com/rs/zerolog"
)

var cfgFile string

// newScanCmd builds and returns the "scan" subcommand.
func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Perform web content discovery against one or more targets",
		Long: `Perform web content discovery using wordlist-based enumeration.

Examples:
  webhound scan --url https://example.com
  webhound scan --url https://example.com --bearer-token TOKEN
  webhound scan --url https://example.com --recursive --max-depth 3
  webhound scan --file targets.txt --threads 20 --rate 10
  webhound scan --url https://example.com --interactive-recursion
  webhound scan --url https://example.com -o report.html`,
		SilenceUsage: true,
	}

	cfg := bindScanFlags(cmd)

	cmd.Flags().StringVar(&cfgFile, "config", "", "Config file path (YAML)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Merge config file if provided.
		if err := loadConfigFile(cfgFile, cfg); err != nil {
			return fmt.Errorf("loading config file: %w", err)
		}

		// Auto-generate output path when --output is not set.
		if cfg.Output == "" {
			cfg.Output = defaultOutputPath(cfg.URL, cfg.File)
			cfg.Format = "txt"
		}

		// Infer output format.
		inferFormat(cfg)

		// Set log level.
		switch {
		case cfg.Verbose:
			logger.SetVerbose()
		case cfg.Quiet:
			logger.SetQuiet()
		default:
			logger.SetLevel(zerolog.InfoLevel)
		}

		// Validate config.
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "\033[1;31m[✗]\033[0m %v\n", err)
			return err
		}

		// Print legal notice.
		if !cfg.Quiet {
			fmt.Fprintln(os.Stderr, "\033[90m[!] Authorized security testing only. Ensure you have written permission.\033[0m")
		}

		// Build and run orchestrator.
		o, err := engine.New(cfg)
		if err != nil {
			return fmt.Errorf("initializing scanner: %w", err)
		}

		return o.Run(context.Background())
	}

	return cmd
}

// defaultOutputPath builds webhoundresults/{host}-{timestamp}.txt from the
// first target URL (or the targets file name when --file is used).
func defaultOutputPath(rawURL, file string) string {
	target := rawURL
	if target == "" {
		target = file
	}
	// Try to extract just the hostname from a full URL.
	if u, err := url.Parse(target); err == nil && u.Hostname() != "" {
		target = u.Hostname()
	} else {
		// Fallback: strip scheme manually and cut at first slash.
		target = strings.TrimPrefix(target, "https://")
		target = strings.TrimPrefix(target, "http://")
		if i := strings.IndexByte(target, '/'); i != -1 {
			target = target[:i]
		}
	}
	// Sanitize any remaining characters that are unsafe in filenames.
	target = strings.NewReplacer(":", "-", "\\", "-").Replace(target)
	if target == "" {
		target = "scan"
	}
	ts := time.Now().Format("2006-01-02-150405")
	return filepath.Join("webhoundresults", target+"-"+ts+".txt")
}
