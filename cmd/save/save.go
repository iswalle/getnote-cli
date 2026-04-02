package save

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/olekukonko/tablewriter"
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

			// Async task: poll until done (pollTask handles JSON mode)
			if taskID, ok := resp.Data.(map[string]interface{}); ok {
				if id, ok := taskID["task_id"].(string); ok && id != "" {
					return pollTask(cmd, c, id)
				}
			}

			// Sync save
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note saved.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Note title")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")
	return cmd
}

// pollTask polls the task status until done, failed, or timeout.
// In JSON mode it runs silently and outputs the final result as JSON.
func pollTask(cmd *cobra.Command, c *client.Client, taskID string) error {
	isJSON := outputFormat(cmd) == "json"
	out := cmd.OutOrStdout()

	if !isJSON {
		fmt.Fprintf(out, "✓ Saving... (task_id: %s)\n", taskID)
	}

	const (
		interval   = 1500 * time.Millisecond
		maxRetries = 20
	)

	var lastResp *client.NoteTaskResponse
	for i := 0; i < maxRetries; i++ {
		time.Sleep(interval)
		if !isJSON {
			fmt.Fprint(out, ".")
		}

		resp, err := c.NoteTask(taskID)
		if err != nil {
			if !isJSON {
				fmt.Fprintln(out, "")
			}
			return err
		}
		lastResp = resp

		switch resp.Data.Status {
		case "done":
			if !isJSON {
				fmt.Fprintln(out, " done")
			}
			if resp.Data.NoteID == "" {
				if isJSON {
					enc := json.NewEncoder(out)
					enc.SetIndent("", "  ")
					return enc.Encode(resp)
				}
				fmt.Fprintln(out, "✓ Note saved.")
				return nil
			}
			noteResp, err := c.NoteGet(resp.Data.NoteID)
			if err != nil {
				return err
			}
			if isJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(noteResp)
			}
			renderNote(cmd, noteResp.Data.Note)
			return nil
		case "failed":
			if isJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(out, "")
			fmt.Fprintf(out, "✗ Failed: %s\n", resp.Data.Msg)
			return nil
		}
		// pending / processing — keep polling
	}

	// Timeout
	if isJSON && lastResp != nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(lastResp)
	}
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "⚠ Timeout. Check later: getnote task %s\n", taskID)
	return nil
}

// renderNote prints a note as a table, mirroring cmd/note/note.go.
func renderNote(cmd *cobra.Command, n client.Note) {
	out := cmd.OutOrStdout()
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Field", "Value"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.Append([]string{"ID", n.NoteID.String()})
	table.Append([]string{"Title", n.Title})
	table.Append([]string{"Type", n.NoteType})
	table.Append([]string{"Created", n.CreatedAt})
	if n.WebPage != nil && n.WebPage.URL != "" {
		table.Append([]string{"URL", n.WebPage.URL})
	}
	if n.WebPage != nil && n.WebPage.Excerpt != "" {
		table.Append([]string{"Excerpt", ui.Truncate(n.WebPage.Excerpt, 120)})
	}
	if n.Content != "" {
		table.Append([]string{"Content", ui.Truncate(n.Content, 200)})
	}
	if tags := n.TagNames(); len(tags) > 0 {
		table.Append([]string{"Tags", strings.Join(tags, ", ")})
	}
	table.Render()
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
