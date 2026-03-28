package auth

import (
	"fmt"
	"os"

	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewAuthCmd returns the auth command tree.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  "Log in, log out, or check the status of your getnote API key.",
	}

	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	return cmd
}

func newLoginCmd() *cobra.Command {
	var apiKey, clientID string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save your API key",
		Long:  "Save an API key (and optional Client ID) to ~/.getnote/config.json for future commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return fmt.Errorf("--api-key is required")
			}

			cfg := config.Get()
			cfg.APIKey = apiKey
			if clientID != "" {
				cfg.ClientID = clientID
			} else if cfg.ClientID == "" {
				cfg.ClientID = config.DefaultClientID
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Logged in successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to save (required)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID to save (optional, defaults to getnote-cli)")
	_ = cmd.MarkFlagRequired("api-key")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()
			if err := cfg.Clear(); err != nil {
				return fmt.Errorf("clearing config: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out successfully.")
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication status",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.Get()

			// Check env var first
			if envKey := os.Getenv("GETNOTE_API_KEY"); envKey != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Authenticated via GETNOTE_API_KEY environment variable.")
				return
			}

			if cfg.IsLoggedIn() {
				masked := maskKey(cfg.APIKey)
				fmt.Fprintf(cmd.OutOrStdout(), "Authenticated. API key: %s\n", masked)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Not authenticated. Run: getnote auth login --api-key <key>")
			}
		},
	}
}

// maskKey returns the key with all but the last 4 characters replaced by *.
func maskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	masked := make([]byte, len(key))
	for i := 0; i < len(key)-4; i++ {
		masked[i] = '*'
	}
	copy(masked[len(key)-4:], key[len(key)-4:])
	return string(masked)
}
