package note

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewNoteCmd returns the note command (detail + update + delete).
func NewNoteCmd() *cobra.Command {
	var field string

	cmd := &cobra.Command{
		Use:   "note <id>",
		Short: "查看笔记详情 / Show note details",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote note 1234567890
  getnote note 1234567890 --field content
  getnote note 1234567890 --field url`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.NoteGet(args[0])
			if err != nil {
				return err
			}

			// --field: output single field as plain text
			if field != "" {
				return printField(resp.Data.Note, field)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			n := resp.Data.Note
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			table.Append([]string{"ID", n.NoteID.String()})
			table.Append([]string{"Title", n.Title})
			table.Append([]string{"Type", n.NoteType})
			table.Append([]string{"Created", n.CreatedAt})
			table.Append([]string{"Updated", n.UpdatedAt})
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
			return nil
		},
	}

	cmd.Flags().StringVar(&field, "field", "", "Output a single field value (id, title, content, type, created_at, updated_at, url, excerpt)")
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	return cmd
}

// printField outputs a single field from a note as plain text.
func printField(n client.Note, field string) error {
	var val string
	switch field {
	case "id":
		val = n.NoteID.String()
	case "title":
		val = n.Title
	case "content":
		val = n.Content
	case "type":
		val = n.NoteType
	case "created_at":
		val = n.CreatedAt
	case "updated_at":
		val = n.UpdatedAt
	case "url":
		if n.WebPage != nil {
			val = n.WebPage.URL
		}
	case "excerpt":
		if n.WebPage != nil {
			val = n.WebPage.Excerpt
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown field %q; valid: id, title, content, type, created_at, updated_at, url, excerpt\n", field)
		os.Exit(1)
	}
	fmt.Println(val)
	return nil
}

func newUpdateCmd() *cobra.Command {
	var title, content, tags string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "更新笔记标题/内容/标签 / Update a note's title, content, or tags",
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
				for i := range req.Tags {
					req.Tags[i] = strings.TrimSpace(req.Tags[i])
				}
			}
			if req.Title == "" && req.Content == "" && req.Tags == nil {
				return fmt.Errorf("at least one of --title, --content, --tag is required")
			}

			c := client.New("")
			resp, err := c.NoteUpdate(req)
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&content, "content", "", "New content (plain_text notes only)")
	cmd.Flags().StringVar(&tags, "tag", "", "Tags (comma-separated, replaces existing)")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "删除笔记（移入回收站）/ Delete a note (moves to trash)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete note %s? [y/N] ", args[0])
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			c := client.New("")
			resp, err := c.NoteDelete(args[0])
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note moved to trash.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}


func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

