// Package client provides an HTTP client for the getnote API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/iswalle/getnote-cli/internal/config"
)

// APIError represents an error returned by the API.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client is an HTTP client for the getnote API.
type Client struct {
	baseURL    string
	apiKey     string
	clientID   string
	httpClient *http.Client
}

// New creates a new API client. It resolves the base URL in priority order:
//  1. GETNOTE_API_URL environment variable
//  2. envTarget ("dev" uses a dev URL)
//  3. Default production URL
func New(envTarget string) *Client {
	baseURL := config.DefaultAPIBaseURL
	if v := os.Getenv("GETNOTE_API_URL"); v != "" {
		baseURL = v
	} else if envTarget == "dev" {
		baseURL = "https://openapi-dev.biji.com"
	}

	cfg := config.Get()

	// API key priority: flag (passed via root cmd override) > env var > config file
	apiKey := cfg.APIKey
	if v := os.Getenv("GETNOTE_API_KEY"); v != "" {
		apiKey = v
	}

	// Client ID priority: env var > config file > default
	clientID := cfg.ClientID
	if v := os.Getenv("GETNOTE_CLIENT_ID"); v != "" {
		clientID = v
	}

	return &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Note API
// ---------------------------------------------------------------------------

// NoteTag represents a tag on a note.
type NoteTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // system | manual | ai
}

// Note represents a single note item.
// Tags can be []string (kb/notes API) or []NoteTag (note/list API);
// use TagNames() to get plain tag names regardless of format.
type Note struct {
	ID        json.Number       `json:"id"`
	NoteID    json.Number       `json:"note_id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	NoteType  string            `json:"note_type"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
	Tags      []json.RawMessage `json:"tags"`
	WebPage   *struct {
		URL     string `json:"url"`
		Excerpt string `json:"excerpt"`
	} `json:"web_page,omitempty"`
}

// TagNames returns tag names regardless of whether tags are strings or objects.
func (n *Note) TagNames() []string {
	var names []string
	for _, raw := range n.Tags {
		// try string first
		var s string
		if json.Unmarshal(raw, &s) == nil {
			names = append(names, s)
			continue
		}
		// try object
		var t NoteTag
		if json.Unmarshal(raw, &t) == nil && t.Name != "" {
			names = append(names, t.Name)
		}
	}
	return names
}

// NoteListData is the data field of the note list response.
type NoteListData struct {
	Notes      []Note      `json:"notes"`
	HasMore    bool        `json:"has_more"`
	NextCursor json.Number `json:"next_cursor"`
	Total      int         `json:"total"`
}

// NoteListParams holds parameters for listing notes.
type NoteListParams struct {
	Limit   int
	SinceID string
}

// NoteListResponse is the response from the note list endpoint.
type NoteListResponse struct {
	Success bool         `json:"success"`
	Data    NoteListData `json:"data"`
}

// NoteList fetches a list of notes.
// GET /open/api/v1/resource/note/list
func (c *Client) NoteList(params NoteListParams) (*NoteListResponse, error) {
	q := url.Values{}
	sinceID := "0"
	if params.SinceID != "" {
		sinceID = params.SinceID
	}
	q.Set("since_id", sinceID)
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	return doGet[NoteListResponse](c, "/open/api/v1/resource/note/list", q)
}

// NoteGetData is the data field of the note detail response.
type NoteGetData struct {
	Note Note `json:"note"`
}

// NoteGetResponse is the response from the note detail endpoint.
type NoteGetResponse struct {
	Success bool        `json:"success"`
	Data    NoteGetData `json:"data"`
}

// NoteGet fetches a single note by ID.
// GET /open/api/v1/resource/note/detail?id=<note_id>
func (c *Client) NoteGet(noteID string) (*NoteGetResponse, error) {
	q := url.Values{"id": {noteID}}
	return doGet[NoteGetResponse](c, "/open/api/v1/resource/note/detail", q)
}

// NoteSaveRequest is the request body for saving a note.
type NoteSaveRequest struct {
	NoteType  string   `json:"note_type"`            // plain_text | link | img_text
	Content   string   `json:"content,omitempty"`    // for plain_text
	LinkURL   string   `json:"link_url,omitempty"`   // for link
	ImageURLs []string `json:"image_urls,omitempty"` // for img_text
	Title     string   `json:"title,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// NoteSaveResponse is the response from the note save endpoint.
type NoteSaveResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// NoteSave saves a new note (URL or plain text).
// POST /open/api/v1/resource/note/save
func (c *Client) NoteSave(req NoteSaveRequest) (*NoteSaveResponse, error) {
	return doPost[NoteSaveResponse](c, "/open/api/v1/resource/note/save", req)
}

// NoteUpdateRequest is the request body for updating a note.
type NoteUpdateRequest struct {
	ID      string   `json:"id"`
	Title   string   `json:"title,omitempty"`
	Content string   `json:"content,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// NoteUpdateResponse is the response from the note update endpoint.
type NoteUpdateResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// NoteUpdate updates an existing note.
// POST /open/api/v1/resource/note/update
func (c *Client) NoteUpdate(req NoteUpdateRequest) (*NoteUpdateResponse, error) {
	return doPost[NoteUpdateResponse](c, "/open/api/v1/resource/note/update", req)
}

// NoteDeleteRequest is the request body for deleting a note.
type NoteDeleteRequest struct {
	ID string `json:"id"`
}

// NoteDeleteResponse is the response from the note delete endpoint.
type NoteDeleteResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// NoteDelete deletes a note by ID.
// POST /open/api/v1/resource/note/delete
func (c *Client) NoteDelete(noteID string) (*NoteDeleteResponse, error) {
	return doPost[NoteDeleteResponse](c, "/open/api/v1/resource/note/delete", NoteDeleteRequest{ID: noteID})
}

// NoteTaskRequest is the request body for querying task progress.
type NoteTaskRequest struct {
	TaskID string `json:"task_id"`
}

// NoteTaskData is the data field of the task progress response.
type NoteTaskData struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`   // pending | processing | done | success | failed
	NoteID string `json:"note_id"`
	Msg    string `json:"msg"`
}

// NoteTaskResponse is the response from the task progress endpoint.
type NoteTaskResponse struct {
	Success bool         `json:"success"`
	Data    NoteTaskData `json:"data"`
}

// NoteTask queries the progress of a note-save task.
// POST /open/api/v1/resource/note/task/progress
func (c *Client) NoteTask(taskID string) (*NoteTaskResponse, error) {
	return doPost[NoteTaskResponse](c, "/open/api/v1/resource/note/task/progress", NoteTaskRequest{TaskID: taskID})
}

// ---------------------------------------------------------------------------
// Knowledge Base API
// ---------------------------------------------------------------------------

// KBTopic represents a single knowledge base.
type KBTopic struct {
	ID          string      `json:"id"`
	TopicID     string      `json:"topic_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Cover       string      `json:"cover"`
	Scope       string      `json:"scope"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	Stats       KBTopicStats `json:"stats"`
}

// KBTopicStats holds knowledge base statistics.
type KBTopicStats struct {
	NoteCount    int `json:"note_count"`
	FileCount    int `json:"file_count"`
	BloggerCount int `json:"blogger_count"`
	LiveCount    int `json:"live_count"`
}

// KBListData is the data field of the knowledge base list response.
type KBListData struct {
	Topics  []KBTopic `json:"topics"`
	HasMore bool      `json:"has_more"`
	Total   int       `json:"total"`
}

// KBListResponse is the response from the knowledge base list endpoint.
type KBListResponse struct {
	Success bool       `json:"success"`
	Data    KBListData `json:"data"`
}

// KBList fetches all knowledge bases.
// GET /open/api/v1/resource/knowledge/list
func (c *Client) KBList() (*KBListResponse, error) {
	return doGet[KBListResponse](c, "/open/api/v1/resource/knowledge/list", url.Values{"page": {"1"}})
}

// KBSubscribedList fetches knowledge bases the user has subscribed to.
// GET /open/api/v1/resource/knowledge/subscribe/list
func (c *Client) KBSubscribedList(page int) (*KBListResponse, error) {
	return doGet[KBListResponse](c, "/open/api/v1/resource/knowledge/subscribe/list", url.Values{"page": {fmt.Sprintf("%d", page)}})
}

// KBCreateRequest is the request body for creating a knowledge base.
type KBCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// KBCreateResponse is the response from the knowledge base create endpoint.
type KBCreateResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// KBCreate creates a new knowledge base.
// POST /open/api/v1/resource/knowledge/create
func (c *Client) KBCreate(req KBCreateRequest) (*KBCreateResponse, error) {
	return doPost[KBCreateResponse](c, "/open/api/v1/resource/knowledge/create", req)
}

// KBNotesParams holds parameters for listing notes in a knowledge base.
type KBNotesParams struct {
	TopicID string
	Limit   int
	Page    int
}

// KBNotesResponse is the response from the knowledge base notes endpoint.
type KBNotesResponse struct {
	Success bool         `json:"success"`
	Data    NoteListData `json:"data"`
}

// KBNotes fetches notes in a knowledge base.
// GET /open/api/v1/resource/knowledge/notes
func (c *Client) KBNotes(params KBNotesParams) (*KBNotesResponse, error) {
	page := 1
	if params.Page > 0 {
		page = params.Page
	}
	q := url.Values{"topic_id": {params.TopicID}, "page": {fmt.Sprintf("%d", page)}}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	return doGet[KBNotesResponse](c, "/open/api/v1/resource/knowledge/notes", q)
}

// KBNotesBatchAddRequest is the request body for batch-adding notes to a KB.
type KBNotesBatchAddRequest struct {
	TopicID string   `json:"topic_id"`
	NoteIDs []string `json:"note_ids"`
}

// KBNotesBatchAddResponse is the response from the batch-add endpoint.
type KBNotesBatchAddResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// KBNotesAdd adds notes to a knowledge base.
// POST /open/api/v1/resource/knowledge/note/batch-add
func (c *Client) KBNotesAdd(topicID string, noteIDs []string) (*KBNotesBatchAddResponse, error) {
	req := KBNotesBatchAddRequest{TopicID: topicID, NoteIDs: noteIDs}
	return doPost[KBNotesBatchAddResponse](c, "/open/api/v1/resource/knowledge/note/batch-add", req)
}

// KBNotesRemoveRequest is the request body for removing notes from a KB.
type KBNotesRemoveRequest struct {
	TopicID string   `json:"topic_id"`
	NoteIDs []string `json:"note_ids"`
}

// KBNotesRemoveResponse is the response from the remove endpoint.
type KBNotesRemoveResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// KBNotesRemove removes notes from a knowledge base.
// POST /open/api/v1/resource/knowledge/note/remove
func (c *Client) KBNotesRemove(topicID string, noteIDs []string) (*KBNotesRemoveResponse, error) {
	req := KBNotesRemoveRequest{TopicID: topicID, NoteIDs: noteIDs}
	return doPost[KBNotesRemoveResponse](c, "/open/api/v1/resource/knowledge/note/remove", req)
}

// ---------------------------------------------------------------------------
// Search (Recall) API
// ---------------------------------------------------------------------------

// RecallResult is a single search result item.
type RecallResult struct {
	NoteID    string `json:"note_id"`
	NoteType  string `json:"note_type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// NoteSearchRequest is the request body for global recall.
type NoteSearchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

// NoteSearchResponse is the response from the global recall endpoint.
type NoteSearchResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Results []RecallResult `json:"results"`
	} `json:"data"`
	Results []RecallResult `json:"results"` // fallback for flat response
}

// NoteSearch performs global semantic search across all notes.
// POST /open/api/v1/resource/recall
func (c *Client) NoteSearch(query string, topK int) (*NoteSearchResponse, error) {
	return doPost[NoteSearchResponse](c, "/open/api/v1/resource/recall", NoteSearchRequest{Query: query, TopK: topK})
}

// KBSearchRequest is the request body for knowledge base recall.
type KBSearchRequest struct {
	TopicID string `json:"topic_id"`
	Query   string `json:"query"`
	TopK    int    `json:"top_k,omitempty"`
}

// KBSearch performs semantic search within a specific knowledge base.
// POST /open/api/v1/resource/recall/knowledge
func (c *Client) KBSearch(topicID, query string, topK int) (*NoteSearchResponse, error) {
	return doPost[NoteSearchResponse](c, "/open/api/v1/resource/recall/knowledge", KBSearchRequest{TopicID: topicID, Query: query, TopK: topK})
}

// ---------------------------------------------------------------------------
// KB Blogger & Live API
// ---------------------------------------------------------------------------

// KBBlogger represents a subscribed blogger in a knowledge base.
type KBBlogger struct {
	FollowID    json.Number `json:"follow_id"`
	AccountName string      `json:"account_name"`
	AccountIcon string      `json:"account_icon"`
	Platform    string      `json:"platform"`
	AccountURL  string      `json:"account_url"`
	FollowTime  string      `json:"follow_time"`
}

// KBBloggerListData is the data field of the blogger list response.
type KBBloggerListData struct {
	Bloggers []KBBlogger `json:"bloggers"`
	HasMore  bool        `json:"has_more"`
	Total    int         `json:"total"`
}

// KBBloggerListResponse is the response from the blogger list endpoint.
type KBBloggerListResponse struct {
	Success bool              `json:"success"`
	Data    KBBloggerListData `json:"data"`
}

// KBBloggerList fetches bloggers subscribed in a knowledge base.
// GET /open/api/v1/resource/knowledge/bloggers
func (c *Client) KBBloggerList(topicID string, page int) (*KBBloggerListResponse, error) {
	return doGet[KBBloggerListResponse](c, "/open/api/v1/resource/knowledge/bloggers", url.Values{
		"topic_id": {topicID},
		"page":     {fmt.Sprintf("%d", page)},
	})
}

// KBBloggerContent represents a content item from a blogger.
type KBBloggerContent struct {
	PostIDAlias  string `json:"post_id_alias"`
	PostTitle    string `json:"post_title"`
	PostSummary  string `json:"post_summary"`
	PostType     string `json:"post_type"`
	PublishTime  string `json:"publish_time"`
	AccountName  string `json:"account_name"`
}

// KBBloggerContentListData is the data field of the blogger content list response.
type KBBloggerContentListData struct {
	Contents []KBBloggerContent `json:"contents"`
	HasMore  bool               `json:"has_more"`
	Total    int                `json:"total"`
}

// KBBloggerContentListResponse is the response from the blogger content list endpoint.
type KBBloggerContentListResponse struct {
	Success bool                     `json:"success"`
	Data    KBBloggerContentListData `json:"data"`
}

// KBBloggerContentList fetches content from a specific blogger in a knowledge base.
// GET /open/api/v1/resource/knowledge/blogger/contents
func (c *Client) KBBloggerContentList(topicID, followID string, page int) (*KBBloggerContentListResponse, error) {
	return doGet[KBBloggerContentListResponse](c, "/open/api/v1/resource/knowledge/blogger/contents", url.Values{
		"topic_id":  {topicID},
		"follow_id": {followID},
		"page":      {fmt.Sprintf("%d", page)},
	})
}

// KBBloggerContentDetail represents the full detail of a blogger content item.
// API returns fields directly in data (flat, not nested under "content").
type KBBloggerContentDetail struct {
	PostIDAlias   string `json:"post_id_alias"`
	PostName      string `json:"post_name"`
	PostTitle     string `json:"post_title"`
	PostType      interface{} `json:"post_type"`
	PostSummary   string `json:"post_summary"`
	PostMediaText string `json:"post_media_text"`
	PostSubtitle  string `json:"post_subtitle"`
	PostURL       string `json:"post_url"`
	PublishTime   string `json:"post_publish_time"`
	CreateTime    string `json:"post_create_time"`
}

// KBBloggerContentDetailResponse is the response from the blogger content detail endpoint.
type KBBloggerContentDetailResponse struct {
	Success bool                   `json:"success"`
	Data    KBBloggerContentDetail `json:"data"`
}

// KBBloggerContentGet fetches full detail of a blogger content item.
// GET /open/api/v1/resource/knowledge/blogger/content/detail
func (c *Client) KBBloggerContentGet(topicID, postID string) (*KBBloggerContentDetailResponse, error) {
	return doGet[KBBloggerContentDetailResponse](c, "/open/api/v1/resource/knowledge/blogger/content/detail", url.Values{
		"topic_id": {topicID},
		"post_id":  {postID},
	})
}

// KBLive represents a live session in a knowledge base.
type KBLive struct {
	LiveID string `json:"live_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// KBLiveListData is the data field of the live list response.
type KBLiveListData struct {
	Lives   []KBLive `json:"lives"`
	HasMore bool     `json:"has_more"`
	Total   int      `json:"total"`
}

// KBLiveListResponse is the response from the live list endpoint.
type KBLiveListResponse struct {
	Success bool           `json:"success"`
	Data    KBLiveListData `json:"data"`
}

// KBLiveList fetches completed live sessions in a knowledge base.
// GET /open/api/v1/resource/knowledge/lives
func (c *Client) KBLiveList(topicID string, page int) (*KBLiveListResponse, error) {
	return doGet[KBLiveListResponse](c, "/open/api/v1/resource/knowledge/lives", url.Values{
		"topic_id": {topicID},
		"page":     {fmt.Sprintf("%d", page)},
	})
}

// KBLiveDetail represents the full detail of a live session.
// The API returns fields directly in data (same shape as blogger content detail).
type KBLiveDetail struct {
	PostName      string `json:"post_name"`
	PostTitle     string `json:"post_title"`
	PostSubtitle  string `json:"post_subtitle"`
	PostSummary   string `json:"post_summary"`
	PostMediaText string `json:"post_media_text"`
	PublishTime   string `json:"post_publish_time"`
}

// KBLiveDetailResponse is the response from the live detail endpoint.
type KBLiveDetailResponse struct {
	Success bool         `json:"success"`
	Data    KBLiveDetail `json:"data"`
}

// KBLiveGet fetches full detail of a live session including summary and transcript.
// GET /open/api/v1/resource/knowledge/live/detail
func (c *Client) KBLiveGet(topicID, liveID string) (*KBLiveDetailResponse, error) {
	return doGet[KBLiveDetailResponse](c, "/open/api/v1/resource/knowledge/live/detail", url.Values{
		"topic_id": {topicID},
		"live_id":  {liveID},
	})
}

// ---------------------------------------------------------------------------
// Quota API
// ---------------------------------------------------------------------------

// QuotaBucket holds used/limit/remaining/reset for one time window.
type QuotaBucket struct {
	Limit     int   `json:"limit"`
	Used      int   `json:"used"`
	Remaining int   `json:"remaining"`
	ResetAt   int64 `json:"reset_at"`
}

// QuotaWindow has daily and monthly buckets.
type QuotaWindow struct {
	Daily   QuotaBucket `json:"daily"`
	Monthly QuotaBucket `json:"monthly"`
}

// QuotaData is the data field of the quota response.
type QuotaData struct {
	Read      QuotaWindow `json:"read"`
	Write     QuotaWindow `json:"write"`
	WriteNote QuotaWindow `json:"write_note"`
}

// QuotaResponse is the response from the quota endpoint.
type QuotaResponse struct {
	Success bool      `json:"success"`
	Data    QuotaData `json:"data"`
}

// QuotaGet fetches the user's API quota usage.
// GET /open/api/v1/resource/rate-limit/quota
func (c *Client) QuotaGet() (*QuotaResponse, error) {
	return doGet[QuotaResponse](c, "/open/api/v1/resource/rate-limit/quota", nil)
}

// ---------------------------------------------------------------------------
// Image Upload API
// ---------------------------------------------------------------------------

// ImageUploadToken holds the credentials for a single OSS upload.
type ImageUploadToken struct {
	Host           string `json:"host"`
	ObjectKey      string `json:"object_key"`
	AccessID       string `json:"accessid"`
	Policy         string `json:"policy"`
	Signature      string `json:"signature"`
	Callback       string `json:"callback"`
	AccessURL      string `json:"access_url"`
	OSSContentType string `json:"oss_content_type"`
}

// ImageUploadTokenData is the data field of the upload token response.
type ImageUploadTokenData struct {
	Tokens []ImageUploadToken `json:"tokens"`
}

// ImageUploadTokenResponse is the response from the upload token endpoint.
type ImageUploadTokenResponse struct {
	Success bool                 `json:"success"`
	Data    ImageUploadTokenData `json:"data"`
}

// ImageGetUploadToken retrieves OSS upload credentials for the given mime type.
// GET /open/api/v1/resource/image/upload_token
func (c *Client) ImageGetUploadToken(mimeType string) (*ImageUploadTokenResponse, error) {
	q := url.Values{"mime_type": {mimeType}, "count": {"1"}}
	return doGet[ImageUploadTokenResponse](c, "/open/api/v1/resource/image/upload_token", q)
}

// ImageUploadToOSS uploads the file at imagePath to OSS using the given token.
// Field order is strictly enforced per OSS signature requirements:
// key → OSSAccessKeyId → policy → signature → callback → Content-Type → file
func (c *Client) ImageUploadToOSS(token ImageUploadToken, imagePath string) error {
	f, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	writeField := func(name, value string) error {
		fw, err := w.CreateFormField(name)
		if err != nil {
			return err
		}
		_, err = fw.Write([]byte(value))
		return err
	}

	if err := writeField("key", token.ObjectKey); err != nil {
		return err
	}
	if err := writeField("OSSAccessKeyId", token.AccessID); err != nil {
		return err
	}
	if err := writeField("policy", token.Policy); err != nil {
		return err
	}
	if err := writeField("signature", token.Signature); err != nil {
		return err
	}
	if err := writeField("callback", token.Callback); err != nil {
		return err
	}
	if err := writeField("Content-Type", token.OSSContentType); err != nil {
		return err
	}

	fw, err := w.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, token.Host, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OSS upload failed: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("OSS upload error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tag API
// ---------------------------------------------------------------------------

// NoteTagsAddRequest is the request body for adding tags to a note.
type NoteTagsAddRequest struct {
	NoteID int64    `json:"note_id"`
	Tags   []string `json:"tags"`
}

// NoteTagsResponseData is the data field of the tags add/list response.
type NoteTagsResponseData struct {
	NoteID string    `json:"note_id"`
	Tags   []NoteTag `json:"tags"`
}

// NoteTagsAddResponse is the response from the tags add endpoint.
type NoteTagsAddResponse struct {
	Success bool                 `json:"success"`
	Data    NoteTagsResponseData `json:"data"`
	Error   *APIError            `json:"error"`
}

// NoteTagsAdd adds tags to a note and returns the updated tag list.
// POST /open/api/v1/resource/note/tags/add
func (c *Client) NoteTagsAdd(noteID int64, tags []string) (*NoteTagsAddResponse, error) {
	return doPost[NoteTagsAddResponse](c, "/open/api/v1/resource/note/tags/add", NoteTagsAddRequest{NoteID: noteID, Tags: tags})
}

// NoteTagsDeleteRequest is the request body for deleting a tag from a note.
type NoteTagsDeleteRequest struct {
	NoteID int64  `json:"note_id"`
	TagID  string `json:"tag_id"`
}

// NoteTagsDeleteResponse is the response from the tags delete endpoint.
type NoteTagsDeleteResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// NoteTagsDelete removes a tag from a note by tag ID.
// POST /open/api/v1/resource/note/tags/delete
func (c *Client) NoteTagsDelete(noteID int64, tagID string) (*NoteTagsDeleteResponse, error) {
	return doPost[NoteTagsDeleteResponse](c, "/open/api/v1/resource/note/tags/delete", NoteTagsDeleteRequest{NoteID: noteID, TagID: tagID})
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	u := c.baseURL + path
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Client-ID", c.clientID)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func doGet[T any](c *Client, path string, query url.Values) (*T, error) {
	fullPath := path
	if len(query) > 0 {
		fullPath += "?" + query.Encode()
	}
	req, err := c.newRequest(http.MethodGet, fullPath, nil)
	if err != nil {
		return nil, err
	}
	return doRequest[T](c, req)
}

func doPost[T any](c *Client, path string, payload interface{}) (*T, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(http.MethodPost, path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return doRequest[T](c, req)
}

func doRequest[T any](c *Client, req *http.Request) (*T, error) {
	const maxRetries = 3
	backoff := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone the request body for retries (body is consumed on each Do)
		var bodyBytes []byte
		if req.Body != nil && req.Body != http.NoBody {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		// 429: rate limited — back off and retry
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt == maxRetries {
				return nil, fmt.Errorf("rate limited after %d retries: %s", maxRetries, string(body))
			}
			time.Sleep(backoff)
			backoff *= 2 // exponential: 1s → 2s → 4s
			// restore body for next attempt
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var result T
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return &result, nil
	}
	return nil, fmt.Errorf("request failed after retries")
}
