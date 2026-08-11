package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestAllCommandsProvideCompleteHelp prevents a release from adding a command
// that cannot explain its usage to users and AI agents through -h/--help.
func TestAllCommandsProvideCompleteHelp(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command != rootCmd && !command.Hidden {
			if strings.TrimSpace(command.Short) == "" {
				t.Errorf("%s: missing Short description", command.CommandPath())
			}
			if command.Runnable() && strings.TrimSpace(command.Example) == "" {
				t.Errorf("%s: runnable command missing Example", command.CommandPath())
			}

			var output bytes.Buffer
			command.InitDefaultHelpFlag()
			command.SetOut(&output)
			command.SetErr(&output)
			if err := command.Help(); err != nil {
				t.Errorf("%s: help failed: %v", command.CommandPath(), err)
			} else {
				help := output.String()
				if !strings.Contains(help, "Usage:") {
					t.Errorf("%s: help missing Usage", command.CommandPath())
				}
				if !strings.Contains(help, "-h, --help") {
					t.Errorf("%s: help missing -h/--help flag", command.CommandPath())
				}
			}
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}

	visit(rootCmd)
}
