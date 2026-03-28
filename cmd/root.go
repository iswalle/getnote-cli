package cmd

import (
	"fmt"
	"os"

	"github.com/iswalle/getnote-cli/cmd/auth"
	"github.com/iswalle/getnote-cli/cmd/kb"
	"github.com/iswalle/getnote-cli/cmd/note"
	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	apiKey    string
	output    string
	envTarget string
)

var rootCmd = &cobra.Command{
	Use:   "getnote",
	Short: "CLI tool for Get笔记 (getnote)",
	Long: `getnote is a command-line tool for interacting with Get笔记.
It allows both humans and AI agents to manage notes and knowledge bases
from the terminal.`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides config and GETNOTE_API_KEY env var)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table or json")
	rootCmd.PersistentFlags().StringVar(&envTarget, "env", "prod", "Environment: prod or dev")

	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(note.NewNoteCmd())
	rootCmd.AddCommand(kb.NewKbCmd())
}

func initConfig() {
	cfg := config.Get()

	// --api-key flag takes highest priority
	if apiKey != "" {
		cfg.APIKey = apiKey
		return
	}

	// GETNOTE_API_KEY env var takes second priority
	if envKey := os.Getenv("GETNOTE_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
		return
	}

	// Fall back to config file (already loaded by config.Get())
}
