package save

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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
	var imageURLFile string

	cmd := &cobra.Command{
		Use:   "save [url|text|image_path]",
		Short: "保存链接、文本或图片笔记 / Save a URL, text note, or image",
		Args: func(cmd *cobra.Command, args []string) error {
			sources := 0
			if imageURLFile != "" {
				sources++
			}
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
			if imageURLFile != "" {
				req, taskID, err := readImageResume(cmd, imageURLFile, client.NoteSaveRequest{Title: title, Tags: tags, TopicID: topicID, ParentID: parentID, ClientRequestID: idempotencyKey})
				if err != nil {
					return err
				}
				return saveUploadedImage(cmd, client.New(""), req, taskID)
			}
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
	cmd.Flags().StringVar(&imageURLFile, "image-url-file", "", "从私有文件读取图片恢复 JSON 或原 HTTPS 地址；仅续接原操作，不重新上传")
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "从标准输入安全读取长文本 / Read long text from stdin")
	return cmd
}

func readImageResume(cmd *cobra.Command, path string, flags client.NoteSaveRequest) (client.NoteSaveRequest, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return flags, "", err
	}
	content := strings.TrimSpace(string(raw))
	req, taskID := flags, ""
	if strings.HasPrefix(content, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return req, "", fmt.Errorf("无效的图片恢复 JSON")
		}
		if data, ok := object["data"]; ok {
			object = nil
			if err := json.Unmarshal(data, &object); err != nil {
				return req, "", fmt.Errorf("无效的图片恢复 data")
			}
		}
		if image, ok := object["uploaded_image"]; ok {
			object = nil
			if err := json.Unmarshal(image, &object); err != nil {
				return req, "", fmt.Errorf("无效的 uploaded_image 恢复数据")
			}
		}
		var stage, imageURL, key string
		json.Unmarshal(object["stage"], &stage)
		json.Unmarshal(object["url"], &imageURL)
		json.Unmarshal(object["client_request_id"], &key)
		if value, ok := object["task_id"]; ok && json.Unmarshal(value, &taskID) != nil {
			return req, "", fmt.Errorf("无效的原图片任务标识")
		}
		decoder := json.NewDecoder(bytes.NewReader(object["request"]))
		decoder.DisallowUnknownFields()
		req = client.NoteSaveRequest{}
		if decoder.Decode(&req) != nil || stage != "oss_uploaded" || req.NoteType != "img_text" || len(req.ImageURLs) != 1 || req.Content != "" || req.LinkURL != "" || req.ClientRequestID == "" {
			return req, "", fmt.Errorf("图片恢复 JSON 缺少完整原保存请求")
		}
		if imageURL != req.ImageURLs[0] || key != req.ClientRequestID {
			return req, "", fmt.Errorf("图片恢复地址或请求键与原保存请求冲突")
		}
		for _, field := range []struct{ name, supplied, original string }{
			{"title", flags.Title, req.Title}, {"topic-id", flags.TopicID, req.TopicID}, {"parent-id", flags.ParentID, req.ParentID}, {"idempotency-key", flags.ClientRequestID, req.ClientRequestID},
		} {
			if cmd.Flags().Changed(field.name) && field.supplied != field.original {
				return req, "", fmt.Errorf("--%s 与原保存请求冲突", field.name)
			}
		}
		if cmd.Flags().Changed("tag") && !slices.Equal(flags.Tags, req.Tags) {
			return req, "", fmt.Errorf("--tag 与原保存请求冲突")
		}
	} else {
		req.NoteType = "img_text"
		req.ImageURLs = []string{content}
	}
	u, parseErr := url.Parse(req.ImageURLs[0])
	if parseErr != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return req, "", fmt.Errorf("image-url-file 必须包含原图片 HTTPS 地址或完整恢复 JSON")
	}
	if req.ClientRequestID == "" {
		return req, "", fmt.Errorf("续接图片保存必须提供原 --idempotency-key")
	}
	if err := validateIdempotencyKey(req.ClientRequestID); err != nil {
		return req, "", err
	}
	return req, taskID, nil
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
		raw = []byte(strings.Join(args, " "))
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
	if idempotencyKey == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		idempotencyKey = "image-note-" + hex.EncodeToString(b)
	}
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
		return &client.RecoveryError{Err: err, Kind: "image", OperationID: token.OperationID, State: "object_upload_unconfirmed"}
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
	return saveUploadedImage(cmd, c, req, "")
}

func saveUploadedImage(cmd *cobra.Command, c *client.Client, req client.NoteSaveRequest, taskID string) (err error) {
	defer func() {
		if err != nil {
			err = &uploadedImageError{Err: err, URL: req.ImageURLs[0], NoteKey: req.ClientRequestID, TaskID: taskID, Request: req}
		}
	}()
	if taskID != "" {
		return pollTask(cmd, c, taskID)
	}
	resp, err := c.NoteSave(req)
	if err != nil {
		return err
	}

	// Extract task_id
	taskID = extractTaskID(resp.Data)
	if taskID == "" {
		if noteID := extractNoteID(resp.Data); noteID != "" && noteID != "0" {
			noteResp, detailErr := c.NoteGet(noteID)
			if detailErr != nil {
				return detailErr
			}
			return outputFinalNote(cmd, noteResp)
		}
		return fmt.Errorf("保存未返回 note_id 或 task_id，不能确认完成")
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

// Machine-only context lets agents resume the exact save without uploading again.
type uploadedImageError struct {
	Err          error
	URL, NoteKey string
	TaskID       string
	Request      client.NoteSaveRequest
}

func (e *uploadedImageError) Error() string {
	return "图片已上传，保存未确认；保留原图片地址和原请求键，勿重新上传"
}
func (e *uploadedImageError) Unwrap() error { return e.Err }
func (e *uploadedImageError) ErrorData() any {
	return map[string]any{"uploaded_image": map[string]any{"stage": "oss_uploaded", "url": e.URL, "client_request_id": e.NoteKey, "task_id": e.TaskID, "request": e.Request}}
}
