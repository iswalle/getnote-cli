package note

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewNoteCmd returns the note command tree.
func NewNoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Manage notes",
		Long:  "Save, list, get, update, delete, search, and query note tasks.",
	}

	cmd.AddCommand(newSaveCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newTaskCmd())
	return cmd
}

// outputFormat reads the --output flag from the root persistent flags.
func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

// envTarget reads the --env flag from the root persistent flags.
func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}

func newSaveCmd() *cobra.Command {
	var title string
	var tags []string

	cmd := &cobra.Command{
		Use:   "save <url|text>",
		Short: "Save a note",
		Long: `Save a note from a URL or plain text.
If the argument starts with http:// or https://, it is saved as a link note.
Otherwise it is saved as a plain-text note.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			c := client.New(envTarget(cmd))

			req := client.NoteSaveRequest{Title: title, Tags: tags}
			if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
				req.URL = input
			} else {
				req.Text = input
			}

			resp, err := c.NoteSave(req)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Note saved successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Optional title for the note")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag to apply (can be specified multiple times)")
	return cmd
}

func newListCmd() *cobra.Command {
	var limit int
	var sinceID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.NoteList(client.NoteListParams{
				Limit:   limit,
				SinceID: sinceID,
			})
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}

			// Table output
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"ID", "Title", "Type", "Created"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)

			// Render raw JSON data rows if server returns them
			renderDataRows(table, resp.Data)
			table.Render()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes to return")
	cmd.Flags().StringVar(&sinceID, "since-id", "", "Cursor for pagination (note ID)")
	return cmd
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <note_id>",
		Short: "Get note details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.NoteGet(args[0])
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}

			// Table output: print key-value pairs
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			renderKV(table, resp.Data)
			table.Render()
			return nil
		},
	}
}

func newUpdateCmd() *cobra.Command {
	var title, content, tags string

	cmd := &cobra.Command{
		Use:   "update <note_id>",
		Short: "Update a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.NoteUpdateRequest{ID: args[0]}
			if title != "" {
				req.Title = title
			}
			if content != "" {
				req.Content = content
			}
			if tags != "" {
				req.Tags = strings.Split(tags, ",")
			}

			c := client.New(envTarget(cmd))
			resp, err := c.NoteUpdate(req)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Note updated successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&content, "content", "", "New content")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags (replaces existing tags)")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <note_id>",
		Short: "Delete a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID := args[0]

			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete note %s? [y/N] ", noteID)
				reader := bufio.NewReader(os.Stdin)
				line, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(line)) != "y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			c := client.New(envTarget(cmd))
			resp, err := c.NoteDelete(noteID)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Note deleted successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "task <task_id>",
		Short: "Query the progress of a note-save task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.NoteTask(args[0])
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			renderKV(table, resp.Data)
			table.Render()
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func printJSON(cmd *cobra.Command, v interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// renderDataRows tries to render a slice of maps as table rows.
func renderDataRows(table *tablewriter.Table, data interface{}) {
	if data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(b, &rows); err != nil {
		// Fallback: print raw JSON as a single cell
		table.Append([]string{string(b)})
		return
	}
	for _, row := range rows {
		id := fmt.Sprintf("%v", row["id"])
		title := fmt.Sprintf("%v", row["title"])
		noteType := fmt.Sprintf("%v", row["type"])
		created := fmt.Sprintf("%v", row["created_at"])
		table.Append([]string{id, title, noteType, created})
	}
}

// renderKV renders a map as key-value table rows.
func renderKV(table *tablewriter.Table, data interface{}) {
	if data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		table.Append([]string{"data", string(b)})
		return
	}
	for k, v := range m {
		table.Append([]string{k, fmt.Sprintf("%v", v)})
	}
}
