package task

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewTaskCmd returns the top-level task command.
func NewTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "task <task_id>",
		Short: "Check the progress of a save task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.NoteTask(args[0])
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			d := resp.Data
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.Append([]string{"Task ID", d.TaskID})
			table.Append([]string{"Status", d.Status})
			if d.NoteID != "" {
				table.Append([]string{"Note ID", d.NoteID})
				fmt.Fprintf(cmd.OutOrStdout(), "")
			}
			if d.Msg != "" {
				table.Append([]string{"Message", d.Msg})
			}
			table.Render()
			if d.Status == "done" && d.NoteID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\n  View note: getnote note %s\n", d.NoteID)
			}
			return nil
		},
	}
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
