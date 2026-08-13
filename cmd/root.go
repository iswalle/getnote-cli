package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/iswalle/getnote-cli/cmd/auth"
	"github.com/iswalle/getnote-cli/cmd/capabilities"
	"github.com/iswalle/getnote-cli/cmd/doctor"
	"github.com/iswalle/getnote-cli/cmd/kb"
	"github.com/iswalle/getnote-cli/cmd/kbs"
	"github.com/iswalle/getnote-cli/cmd/kbssub"
	"github.com/iswalle/getnote-cli/cmd/note"
	"github.com/iswalle/getnote-cli/cmd/notes"
	"github.com/iswalle/getnote-cli/cmd/quota"
	"github.com/iswalle/getnote-cli/cmd/save"
	"github.com/iswalle/getnote-cli/cmd/search"
	"github.com/iswalle/getnote-cli/cmd/setup"
	"github.com/iswalle/getnote-cli/cmd/tag"
	"github.com/iswalle/getnote-cli/cmd/task"
	"github.com/iswalle/getnote-cli/cmd/update"
	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	apiKey string
	output string
)

var rootCmd = &cobra.Command{
	Use:     "getnote",
	Short:   "Get笔记命令行工具 / CLI tool for Get笔记",
	Version: version.Version,
	Long: `getnote 是 Get笔记的命令行工具，支持保存、搜索、管理笔记和知识库。
适合人工操作和 AI Agent 集成使用。

getnote is a command-line tool for interacting with Get笔记.
It allows both humans and AI agents to manage notes and knowledge bases
from the terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		writeError(os.Stderr, err, output)
		os.Exit(1)
	}
}

func writeError(w io.Writer, err error, format string) {
	var requestErr *client.RequestError
	if format == "json" {
		apiErr := client.APIError{
			Code:      -1,
			Message:   err.Error(),
			Reason:    "cli_error",
			Retryable: false,
		}
		requestID := ""
		if errors.As(err, &requestErr) {
			apiErr = requestErr.APIError
			requestID = requestErr.RequestID
			if apiErr.Code == 10201 {
				apiErr.MembershipURL = client.MembershipPurchaseURL
			}
		}
		payload := struct {
			Success   bool             `json:"success"`
			Data      interface{}      `json:"data"`
			Error     *client.APIError `json:"error"`
			RequestID string           `json:"request_id,omitempty"`
		}{
			Success:   false,
			Data:      nil,
			Error:     &apiErr,
			RequestID: requestID,
		}
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(payload); encodeErr == nil {
			return
		}
	}
	if errors.As(err, &requestErr) && requestErr.Code == 10201 {
		fmt.Fprintf(w, "%s\n开通会员：%s\n", err, client.MembershipPurchaseURL)
		return
	}
	fmt.Fprintln(w, err)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides config and GETNOTE_API_KEY env var)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "输出格式 / Output format: table or json")

	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(capabilities.NewCapabilitiesCmd())
	rootCmd.AddCommand(doctor.NewDoctorCmd())
	rootCmd.AddCommand(setup.NewSetupCmd())
	rootCmd.AddCommand(save.NewSaveCmd())
	rootCmd.AddCommand(task.NewTaskCmd())
	rootCmd.AddCommand(notes.NewNotesCmd())
	rootCmd.AddCommand(note.NewNoteCmd())
	rootCmd.AddCommand(kbs.NewKbsCmd())
	rootCmd.AddCommand(kbssub.NewKbsSubCmd())
	rootCmd.AddCommand(kb.NewKbCmd())
	rootCmd.AddCommand(search.NewSearchCmd())
	rootCmd.AddCommand(tag.NewTagCmd())
	rootCmd.AddCommand(quota.NewQuotaCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(update.NewUpdateCmd())
}

func initConfig() {
	cfg := config.Get()

	if apiKey != "" {
		cfg.APIKey = apiKey
		return
	}

	if envKey := os.Getenv("GETNOTE_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
		return
	}
}
