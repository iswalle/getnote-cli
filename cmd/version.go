package cmd

import (
	"fmt"

	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var checkUpdate bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "getnote version %s\n", version.Version)

			if checkUpdate {
				fmt.Fprintln(cmd.OutOrStdout(), "Checking for updates...")
				if hint := version.CheckUpdate(); hint != "" {
					fmt.Fprintln(cmd.OutOrStdout(), "")
					fmt.Fprintln(cmd.OutOrStdout(), hint)
				} else if version.Version != "dev" {
					fmt.Fprintln(cmd.OutOrStdout(), "Already up to date.")
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkUpdate, "check-update", false, "Check for a newer version on GitHub")
	return cmd
}
