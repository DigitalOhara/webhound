package cli

import (
	"context"
	"fmt"
	"os"

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
