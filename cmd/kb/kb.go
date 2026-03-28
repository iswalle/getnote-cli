package kb

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewKbCmd returns the kb command tree.
func NewKbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kb",
		Short: "Manage knowledge bases",
		Long:  "List, create knowledge bases and manage their notes.",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newNotesCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
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

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all knowledge bases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBList()
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"ID", "Name", "Description", "Notes"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			renderKBRows(table, resp.Data)
			table.Render()
			return nil
		},
	}
}

func newCreateCmd() *cobra.Command {
	var desc string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBCreate(client.KBCreateRequest{
				Name:        args[0],
				Description: desc,
			})
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Knowledge base created successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&desc, "desc", "", "Description for the knowledge base")
	return cmd
}

func newNotesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "notes <topic_id>",
		Short: "List notes in a knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBNotes(client.KBNotesParams{
				TopicID: args[0],
				Limit:   limit,
			})
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"ID", "Title", "Type", "Created"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			renderNoteRows(table, resp.Data)
			table.Render()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes to return")
	return cmd
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <topic_id> <note_id> [note_id...]",
		Short: "Add notes to a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topicID := args[0]
			noteIDs := args[1:]

			c := client.New(envTarget(cmd))
			resp, err := c.KBNotesAdd(topicID, noteIDs)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %d note(s) to knowledge base %s.\n", len(noteIDs), topicID)
			return nil
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <topic_id> <note_id> [note_id...]",
		Short: "Remove notes from a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topicID := args[0]
			noteIDs := args[1:]

			c := client.New(envTarget(cmd))
			resp, err := c.KBNotesRemove(topicID, noteIDs)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				return printJSON(cmd, resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %d note(s) from knowledge base %s.\n", len(noteIDs), topicID)
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

func renderKBRows(table *tablewriter.Table, data interface{}) {
	if data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(b, &rows); err != nil {
		table.Append([]string{string(b)})
		return
	}
	for _, row := range rows {
		id := fmt.Sprintf("%v", row["id"])
		name := fmt.Sprintf("%v", row["name"])
		desc := fmt.Sprintf("%v", row["description"])
		count := fmt.Sprintf("%v", row["note_count"])
		table.Append([]string{id, name, desc, count})
	}
}

func renderNoteRows(table *tablewriter.Table, data interface{}) {
	if data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(b, &rows); err != nil {
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
