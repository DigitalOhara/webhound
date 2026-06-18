package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/session"
)

var rootCmd = &cobra.Command{
	Use:   "webhound",
	Short: "WebHound — Web content discovery and enumeration tool",
	Long: `WebHound is a production-grade web content discovery tool for authorized
security assessments, penetration testing, and bug bounty engagements.

It combines the capabilities of Dirsearch and Gobuster with enhanced safety
controls, session management, and flexible authentication support.`,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newProfileCmd())
	rootCmd.AddCommand(newSessionCmd())
}

// newProfileCmd returns the profile management subcommand.
func newProfileCmd() *cobra.Command {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage authentication profiles",
	}

	profileCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all saved profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := config.ListProfiles()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("No profiles saved.")
				return nil
			}
			for _, n := range names {
				fmt.Println(" •", n)
			}
			return nil
		},
	})

	var (
		profileName  string
		scopePattern string
		bearerToken  string
		basicAuth    string
		cookies      []string
		headers      []string
	)

	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save an authentication profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileName == "" {
				return fmt.Errorf("--name is required")
			}
			p := &config.AuthProfile{
				Name:        profileName,
				Scope:       scopePattern,
				BearerToken: bearerToken,
				BasicAuth:   basicAuth,
				Cookies:     cookies,
				Headers:     make(map[string]string),
			}
			for _, h := range headers {
				parts := splitHeader(h)
				if parts != nil {
					p.Headers[parts[0]] = parts[1]
				}
			}
			if err := config.SaveProfile(p); err != nil {
				return err
			}
			fmt.Printf("\033[1;32m[✓]\033[0m Profile %q saved.\n", profileName)
			return nil
		},
	}
	saveCmd.Flags().StringVar(&profileName, "name", "", "Profile name")
	saveCmd.Flags().StringVar(&scopePattern, "scope", "", "Target scope pattern (e.g. *.example.com)")
	saveCmd.Flags().StringVar(&bearerToken, "bearer-token", "", "Bearer token")
	saveCmd.Flags().StringVar(&basicAuth, "basic-auth", "", "Basic auth (user:pass)")
	saveCmd.Flags().StringArrayVar(&cookies, "cookie", nil, "Cookie(s)")
	saveCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Header(s)")

	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.DeleteProfile(args[0]); err != nil {
				return err
			}
			fmt.Printf("\033[1;32m[✓]\033[0m Profile %q deleted.\n", args[0])
			return nil
		},
	}

	profileCmd.AddCommand(saveCmd, deleteCmd)
	return profileCmd
}

// newSessionCmd returns the session management subcommand.
func newSessionCmd() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage scan sessions",
	}

	sessionCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List saved sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := session.ListSessions()
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}
			for _, id := range ids {
				fmt.Println(" •", id)
			}
			return nil
		},
	})

	return sessionCmd
}

func splitHeader(h string) []string {
	for i, c := range h {
		if c == ':' {
			return []string{
				h[:i],
				h[i+1:],
			}
		}
	}
	return nil
}
