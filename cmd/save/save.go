package save

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	var contentFile string
	var readStdin bool

	cmd := &cobra.Command{
		Use:   "save [url|text|image_path]",
		Short: "保存链接、文本或图片笔记 / Save a URL, text note, or image",
		Args: func(cmd *cobra.Command, args []string) error {
			sources := 0
			if len(args) > 0 {
				sources++
			}
			if contentFile != "" {
				sources++
			}
			if readStdin {
				sources++
			}
			if sources != 1 {
				return fmt.Errorf("请且只提供一种内容来源：命令参数、--content-file 或 --stdin")
			}
			return nil
		},
		Example: `  getnote save https://example.com --title "Great article"
  getnote save "Remember to review the docs" --tag work --tag important
  getnote save ./screenshot.png --title "Design mockup"
  getnote save --content-file ./long-note.md --title "Long note"
  pbpaste | getnote save --stdin --title "Clipboard note"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := resolveContent(cmd, args, contentFile, readStdin)
			if err != nil {
				return err
			}
			if err := validateIdempotencyKey(idempotencyKey); err != nil {
				return err
			}
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

			// Synchronous save. A successful response must identify the created
			// note; otherwise callers cannot safely verify or retry the write.
			if noteID := extractNoteID(resp.Data); noteID != "" && noteID != "0" {
				noteResp, detailErr := c.NoteGet(noteID)
				if detailErr == nil {
					return outputFinalNote(cmd, noteResp)
				}
				if outputFormat(cmd) == "json" {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(resp)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Note saved.\nNote URL: %s\n", c.NoteURL(noteID))
				return nil
			}
			return fmt.Errorf("保存请求未返回 note_id 或 task_id，无法确认是否完成；请勿直接重复保存")
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "笔记标题 / Note title")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "标签，可重复 / Tag (repeatable)")
	cmd.Flags().StringVar(&topicID, "topic-id", "", "存入指定知识库 / Save into this knowledge base")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "创建为指定笔记的子笔记 / Create as a child note")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "重试时复用的幂等键，1-128 位 ASCII / Retry-safe request key")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "从 UTF-8 文件安全读取长文本 / Read long text from a UTF-8 file")
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "从标准输入安全读取长文本 / Read long text from stdin")
	return cmd
}

func resolveContent(cmd *cobra.Command, args []string, contentFile string, readStdin bool) (string, error) {
	var raw []byte
	var err error
	switch {
	case contentFile != "":
		raw, err = os.ReadFile(contentFile)
		if err != nil {
			return "", fmt.Errorf("读取内容文件失败: %w", err)
		}
	case readStdin:
		raw, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("读取标准输入失败: %w", err)
		}
	default:
		return strings.Join(args, " "), nil
	}
	content := string(raw)
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("保存内容不能为空")
	}
	return content, nil
}

func validateIdempotencyKey(key string) error {
	if key == "" {
		return nil
	}
	if len(key) > 128 {
		return fmt.Errorf("--idempotency-key 必须为 1-128 位 ASCII 字符")
	}
	for _, r := range key {
		if r < 0x21 || r > 0x7e {
			return fmt.Errorf("--idempotency-key 必须为 1-128 位可见 ASCII 字符")
		}
	}
	return nil
}

// isImagePath returns true if the arg looks like a local image file path.
func isImagePath(arg string) bool {
	ext := strings.ToLower(filepath.Ext(arg))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return false
	}
	info, err := os.Stat(arg)
	return err == nil && !info.IsDir()
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

func validateImageFormat(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开图片失败: %w", err)
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}
	detected := http.DetectContentType(header[:n])
	if n >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		detected = "image/webp"
	}

	extFormat := mimeTypeFromExt(filepath.Ext(path))
	wantMIME := map[string]string{
		"jpg":  "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
	}[extFormat]
	if detected != wantMIME {
		return "", fmt.Errorf("图片扩展名与实际格式不一致：扩展名为 %s，检测到 %s", extFormat, detected)
	}
	return extFormat, nil
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

	mimeType, err := validateImageFormat(imagePath)
	if err != nil {
		return err
	}

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
		if noteID := extractNoteID(resp.Data); noteID != "" && noteID != "0" {
			noteResp, detailErr := c.NoteGet(noteID)
			if detailErr != nil {
				return fmt.Errorf("图片已提交但读取最终笔记失败: %w", detailErr)
			}
			return outputFinalNote(cmd, noteResp)
		}
		return fmt.Errorf("图片已上传，但保存响应未返回 note_id 或 task_id，无法确认是否完成；请勿直接重复保存")
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

func extractNoteID(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	switch id := m["note_id"].(type) {
	case string:
		return id
	case json.Number:
		return id.String()
	default:
		return ""
	}
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
			if resp.Data.NoteID == "" || resp.Data.NoteID == "0" {
				return fmt.Errorf("任务 %s 已结束但未返回有效 note_id，无法确认保存结果；请勿直接重复保存", taskID)
			}
			noteResp, err := c.NoteGet(resp.Data.NoteID)
			if err != nil {
				return err
			}
			return outputFinalNote(cmd, noteResp)
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

func outputFinalNote(cmd *cobra.Command, resp *client.NoteGetResponse) error {
	if resp == nil {
		return fmt.Errorf("保存完成但未读取到最终笔记")
	}
	noteID := ui.NoteID(resp.Data.Note.NoteID, resp.Data.Note.ID)
	if noteID == "" || noteID == "0" || resp.Data.Note.NoteURL == "" {
		return fmt.Errorf("保存完成但最终结果缺少有效 note_id 或 note_url")
	}
	if outputFormat(cmd) == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	renderNote(cmd, resp.Data.Note)
	return nil
}

// renderNote prints a note as a table, mirroring cmd/note/note.go.
func renderNote(cmd *cobra.Command, n client.Note) {
	out := cmd.OutOrStdout()
	table := tablewriter.NewWriter(out)
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
