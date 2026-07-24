package save

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	var topicID string
	var parentID string
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "save <url|text|image_path>",
		Short: "保存链接、文本或图片笔记 / Save a URL, text note, or image",
		Args:  cobra.MinimumNArgs(1),
		Example: `  getnote save https://example.com --title "Great article"
  getnote save "Remember to review the docs" --tag work --tag important
  getnote save ./screenshot.png --title "Design mockup"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")
			c := client.New("")

			// Detect local image file
			if isImagePath(content) {
				return saveImage(cmd, c, content, title, tags, topicID, parentID, idempotencyKey)
			}

			req := client.NoteSaveRequest{
				Tags:            tags,
				TopicID:         topicID,
				ParentID:        parentID,
				ClientRequestID: idempotencyKey,
			}
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
			if id := extractTaskID(resp.Data); id != "" {
				return pollTask(cmd, c, id)
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
	cmd.Flags().StringVar(&topicID, "topic-id", "", "Save the note into this knowledge base")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Create as a child of this note ID")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Retry-safe request key (1-128 ASCII characters)")
	return cmd
}

// isImagePath returns true if the arg looks like a local image file path.
func isImagePath(arg string) bool {
	if !strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, "./") && !strings.HasPrefix(arg, "../") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(arg))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return false
	}
	_, err := os.Stat(arg)
	return err == nil
}

// mimeTypeFromExt maps image extensions to the mime_type param expected by the API.
func mimeTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "jpg"
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	}
	return "png"
}

// saveImage handles the full image save flow: get token → upload OSS → save note → poll.
func saveImage(
	cmd *cobra.Command,
	c *client.Client,
	imagePath, title string,
	tags []string,
	topicID, parentID, idempotencyKey string,
) error {
	isJSON := outputFormat(cmd) == "json"
	out := cmd.OutOrStdout()

	mimeType := mimeTypeFromExt(filepath.Ext(imagePath))

	// Step 1: get upload token
	tokenResp, err := c.ImageGetUploadToken(mimeType)
	if err != nil {
		return fmt.Errorf("getting upload token: %w", err)
	}
	token := tokenResp.Data
	if token.AccessID == "" || token.Policy == "" {
		return fmt.Errorf("no upload token returned")
	}

	// Step 2: upload to OSS
	if !isJSON {
		fmt.Fprintf(out, "Uploading %s...\n", filepath.Base(imagePath))
	}
	if err := c.ImageUploadToOSS(token, imagePath); err != nil {
		return fmt.Errorf("uploading image: %w", err)
	}

	// Step 3: save img_text note
	req := client.NoteSaveRequest{
		NoteType:        "img_text",
		ImageURLs:       []string{token.AccessURL},
		Title:           title,
		Tags:            tags,
		TopicID:         topicID,
		ParentID:        parentID,
		ClientRequestID: idempotencyKey,
	}
	resp, err := c.NoteSave(req)
	if err != nil {
		return fmt.Errorf("saving image note: %w", err)
	}

	// Extract task_id
	taskID := extractTaskID(resp.Data)
	if taskID == "" {
		if isJSON {
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}
		fmt.Fprintln(out, "✓ Image note saved.")
		return nil
	}
	return pollTask(cmd, c, taskID)
}

// extractTaskID tries to extract a task_id from a save response data value.
// Handles both {task_id: "..."} and {tasks: [{task_id: "..."}]} shapes.
func extractTaskID(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	if id, ok := m["task_id"].(string); ok && id != "" {
		return id
	}
	// Link-type shape: data.tasks[0].task_id
	if tasks, ok := m["tasks"].([]interface{}); ok && len(tasks) > 0 {
		if t, ok := tasks[0].(map[string]interface{}); ok {
			if id, ok := t["task_id"].(string); ok {
				return id
			}
		}
	}
	return ""
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
		interval   = 2000 * time.Millisecond
		maxRetries = 40 // up to ~80s
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
		case "done", "success":
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
			message := resp.Data.ErrorMsg
			if message == "" {
				message = resp.Data.Msg
			}
			if isJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(resp); err != nil {
					return err
				}
				return fmt.Errorf("note task failed: %s", message)
			}
			fmt.Fprintln(out, "")
			fmt.Fprintf(out, "✗ Failed: %s\n", message)
			return fmt.Errorf("note task failed: %s", message)
		}
		// pending / processing — keep polling
	}

	// Timeout
	if isJSON && lastResp != nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(lastResp); err != nil {
			return err
		}
		return fmt.Errorf("note task timed out: %s", taskID)
	}
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "⚠ Timeout. Check later: getnote task %s\n", taskID)
	return fmt.Errorf("note task timed out: %s", taskID)
}

// renderNote prints a note as a table, mirroring cmd/note/note.go.
func renderNote(cmd *cobra.Command, n client.Note) {
	out := cmd.OutOrStdout()
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Field", "Value"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.Append([]string{"ID", ui.NoteID(n.NoteID, n.ID)})
	table.Append([]string{"Title", n.Title})
	table.Append([]string{"Type", n.NoteType})
	table.Append([]string{"Created", n.CreatedAt})
	if n.WebPage != nil && n.WebPage.URL != "" {
		table.Append([]string{"URL", n.WebPage.URL})
	}
	if n.WebPage != nil && n.WebPage.Excerpt != "" {
		table.Append([]string{"Excerpt", ui.Truncate(n.WebPage.Excerpt, 120)})
	}
	if n.WebPage != nil && n.WebPage.Content != "" {
		table.Append([]string{"Web Content", ui.Truncate(n.WebPage.Content, 200)})
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
