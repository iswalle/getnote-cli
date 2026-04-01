package save

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/spf13/cobra"
)

// NewSaveCmd returns the top-level save command.
func NewSaveCmd() *cobra.Command {
	var title string
	var tags []string

	cmd := &cobra.Command{
		Use:   "save <url|text>",
		Short: "Save a URL or text note",
		Args:  cobra.MinimumNArgs(1),
		Example: `  getnote save https://example.com --title "Great article"
  getnote save "Remember to review the docs" --tag work --tag important`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")
			c := client.New(envTarget(cmd))

			req := client.NoteSaveRequest{Tags: tags}
			if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
				req.NoteType = "link"
				req.LinkURL = content
				req.Title = title
			} else {
				req.NoteType = "plain_text"
				req.Content = content
				req.Title = title
			}

			resp, err := c.NoteSave(req)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			// Show task ID for async tasks, or success for sync
			if taskID, ok := resp.Data.(map[string]interface{}); ok {
				if id, ok := taskID["task_id"].(string); ok && id != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Saving... task_id: %s\n  Check progress: getnote task %s\n", id, id)
					return nil
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note saved.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Note title")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
