package note

import (
	"encoding/json"
	"fmt"
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
			table.Append([]string{"ID", ui.NoteID(n.NoteID, n.ID)})
			if n.NoteURL != "" {
				table.Append([]string{"Note URL", n.NoteURL})
			}
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
			if n.Source != "" {
				table.Append([]string{"Source", n.Source})
			}
			if n.ChildrenCount != 0 {
				table.Append([]string{"Children", fmt.Sprintf("%d", n.ChildrenCount)})
			}
			if tags := n.TagNames(); len(tags) > 0 {
				table.Append([]string{"Tags", strings.Join(tags, ", ")})
			}
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&field, "field", "", "Output a single field value (id, note_url, title, content, type, created_at, updated_at, url, excerpt, web_content, audio_original, source, tags)")
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newShareCmd())
	return cmd
}

// printField outputs a single field from a note as plain text.
func printField(n client.Note, field string) error {
	var val string
	switch field {
	case "id":
		val = ui.NoteID(n.NoteID, n.ID)
	case "note_url":
		val = n.NoteURL
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
	case "web_content":
		if n.WebPage != nil {
			val = n.WebPage.Content
		}
	case "audio_original":
		if n.Audio != nil {
			val = n.Audio.Original
		}
	case "source":
		val = n.Source
	case "tags":
		val = strings.Join(n.TagNames(), ", ")
	default:
		return fmt.Errorf("unknown field %q; valid: id, note_url, title, content, type, created_at, updated_at, url, excerpt, web_content, audio_original, source, tags", field)
	}
	fmt.Println(val)
	return nil
}

func newUpdateCmd() *cobra.Command {
	var title, content, tags string
	var yes bool

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "更新笔记标题/内容/标签 / Update a note's title, content, or tags",
		Example: `  getnote note update 1896830231705320746 --title "新标题"
  getnote note update 1896830231705320746 --content "更新后的正文"
  getnote note update 1896830231705320746 --tag "工作,重点"`,
		Args: cobra.ExactArgs(1),
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
			if req.Content != "" || req.Tags != nil {
				approved, err := ui.ConfirmDestructive(cmd, yes, "This replaces existing note content or all tags. Continue?")
				if err != nil {
					return err
				}
				if !approved {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
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
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&content, "content", "", "New content (plain_text notes only)")
	cmd.Flags().StringVar(&tags, "tag", "", "Tags (comma-separated, replaces existing)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm replacing content or all tags without prompting")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "删除笔记（移入回收站）/ Delete a note (moves to trash)",
		Example: `  getnote note delete 1896830231705320746
  getnote note delete 1896830231705320746 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			approved, err := ui.ConfirmDestructive(cmd, yes, fmt.Sprintf("Delete note %s?", args[0]))
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
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
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Note moved to trash.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newShareCmd() *cobra.Command {
	var excludeAudio bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "share <id>",
		Short: "生成公开分享链接 / Create a public share link",
		Long: `Create a publicly accessible share link for a note.
This changes who can view the note and therefore requires confirmation.`,
		Example: `  getnote note share 1896830231705320746
  getnote note share 1896830231705320746 --exclude-audio --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			approved, err := ui.ConfirmDestructive(cmd, yes, fmt.Sprintf("Create a public share link for note %s?", args[0]))
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			resp, err := client.New("").NoteShare(args[0], excludeAudio)
			if err != nil {
				return ui.FriendlyError(err)
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), resp.Data.ShareURL)
			return nil
		},
	}
	cmd.Flags().BoolVar(&excludeAudio, "exclude-audio", false, "Exclude audio from the shared note")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm creating a public share link without prompting")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}
