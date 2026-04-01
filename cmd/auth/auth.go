package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/spf13/cobra"
)

const (
	deviceCodeURL = config.DefaultAPIBaseURL + "/open/api/v1/oauth/device/code"
	tokenURL      = config.DefaultAPIBaseURL + "/open/api/v1/oauth/token"
	oauthClientID = "cli_a1b2c3d4e5f6789012345678abcdef90"
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
		Short: "Authenticate with Get笔记",
		Long: `Authenticate via OAuth Device Flow (recommended) or directly with an API key.

  getnote auth login              # OAuth: open browser to authorize
  getnote auth login --api-key <key>  # Direct: save API key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct mode: --api-key provided
			if apiKey != "" {
				cfg := config.Get()
				cfg.APIKey = apiKey
				if clientID != "" {
					cfg.ClientID = clientID
				} else if cfg.ClientID == "" {
					cfg.ClientID = oauthClientID
				}
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("saving config: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "✅ Logged in successfully.")
				return nil
			}

			// OAuth Device Flow
			return runDeviceFlow(cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (skips OAuth, saves directly)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID (optional, used with --api-key)")
	return cmd
}

// runDeviceFlow runs the OAuth 2.0 Device Authorization Flow.
func runDeviceFlow(out io.Writer) error {
	// Step 1: request device code
	body := fmt.Sprintf(`{"client_id":"%s"}`, oauthClientID)
	resp, err := http.Post(deviceCodeURL, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}
	defer resp.Body.Close()

	var dcResp struct {
		Success bool `json:"success"`
		Data    struct {
			Code            string `json:"code"`
			VerificationURI string `json:"verification_uri"`
			UserCode        string `json:"user_code"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dcResp); err != nil {
		return fmt.Errorf("parsing device code response: %w", err)
	}
	if !dcResp.Success || dcResp.Data.Code == "" {
		return fmt.Errorf("failed to get device code from server")
	}

	d := dcResp.Data
	interval := time.Duration(d.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(d.ExpiresIn) * time.Second)

	// Step 2: show user the link
	fmt.Fprintf(out, "\n🔗 Open this URL to authorize:\n\n   %s\n\n", d.VerificationURI)
	fmt.Fprintf(out, "⚠️  Confirm code on the page: %s\n\n", d.UserCode)
	fmt.Fprintf(out, "Waiting for authorization")

	// Step 3: poll
	pollBody := fmt.Sprintf(`{"grant_type":"device_code","client_id":"%s","code":"%s"}`, oauthClientID, d.Code)
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		fmt.Fprint(out, ".")

		r, err := http.Post(tokenURL, "application/json", strings.NewReader(pollBody))
		if err != nil {
			continue
		}

		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		rawStr := string(raw)

		// pending
		if strings.Contains(rawStr, "authorization_pending") {
			continue
		}
		// terminal errors
		if strings.Contains(rawStr, "rejected") {
			fmt.Fprintln(out)
			return fmt.Errorf("authorization rejected by user")
		}
		if strings.Contains(rawStr, "expired_token") {
			fmt.Fprintln(out)
			return fmt.Errorf("authorization code expired, please try again")
		}
		if strings.Contains(rawStr, "already_consumed") {
			fmt.Fprintln(out)
			return fmt.Errorf("authorization code already used")
		}

		// success
		var tokenResp struct {
			Success bool `json:"success"`
			Data    struct {
				ClientID  string `json:"client_id"`
				APIKey    string `json:"api_key"`
				ExpiresAt int64  `json:"expires_at"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &tokenResp); err == nil && tokenResp.Success && tokenResp.Data.APIKey != "" {
			fmt.Fprintln(out)
			cfg := config.Get()
			cfg.APIKey = tokenResp.Data.APIKey
			cfg.ClientID = tokenResp.Data.ClientID
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			expiry := time.Unix(tokenResp.Data.ExpiresAt, 0).Format("2006-01-02")
			fmt.Fprintf(out, "✅ Authorized! API key valid until %s\n", expiry)
			fmt.Fprintf(out, "   To revoke: https://www.biji.com/openapi?tab=keys\n")
			return nil
		}
	}

	fmt.Fprintln(out)
	return fmt.Errorf("authorization timed out, please try again")
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

			if envKey := os.Getenv("GETNOTE_API_KEY"); envKey != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Authenticated via GETNOTE_API_KEY environment variable.")
				return
			}

			if cfg.IsLoggedIn() {
				masked := maskKey(cfg.APIKey)
				fmt.Fprintf(cmd.OutOrStdout(), "Authenticated. API key: %s\n", masked)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Not authenticated. Run: getnote auth login")
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
