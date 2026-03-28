// Package client provides an HTTP client for the getnote API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/iswalle/getnote-cli/internal/config"
)

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
		baseURL = "https://api-dev.getnote.ai"
	}

	cfg := config.Get()
	return &Client{
		baseURL:  baseURL,
		apiKey:   cfg.APIKey,
		clientID: cfg.ClientID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Note API
// ---------------------------------------------------------------------------

// NoteListParams holds parameters for listing notes.
type NoteListParams struct {
	Limit   int
	SinceID string
}

// NoteListResponse is the response from the note list endpoint.
type NoteListResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteList fetches a list of notes.
// GET /api/v1/resource/note/list
func (c *Client) NoteList(params NoteListParams) (*NoteListResponse, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.SinceID != "" {
		q.Set("since_id", params.SinceID)
	}
	return doGet[NoteListResponse](c, "/api/v1/resource/note/list", q)
}

// NoteGetResponse is the response from the note detail endpoint.
type NoteGetResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteGet fetches a single note by ID.
// GET /api/v1/resource/note/detail?id=<note_id>
func (c *Client) NoteGet(noteID string) (*NoteGetResponse, error) {
	q := url.Values{"id": {noteID}}
	return doGet[NoteGetResponse](c, "/api/v1/resource/note/detail", q)
}

// NoteSaveRequest is the request body for saving a note.
type NoteSaveRequest struct {
	URL     string   `json:"url,omitempty"`
	Text    string   `json:"text,omitempty"`
	Title   string   `json:"title,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// NoteSaveResponse is the response from the note save endpoint.
type NoteSaveResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteSave saves a new note (URL or plain text).
// POST /api/v1/resource/note/save
func (c *Client) NoteSave(req NoteSaveRequest) (*NoteSaveResponse, error) {
	return doPost[NoteSaveResponse](c, "/api/v1/resource/note/save", req)
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
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteUpdate updates an existing note.
// POST /api/v1/resource/note/update
func (c *Client) NoteUpdate(req NoteUpdateRequest) (*NoteUpdateResponse, error) {
	return doPost[NoteUpdateResponse](c, "/api/v1/resource/note/update", req)
}

// NoteDeleteRequest is the request body for deleting a note.
type NoteDeleteRequest struct {
	ID string `json:"id"`
}

// NoteDeleteResponse is the response from the note delete endpoint.
type NoteDeleteResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteDelete deletes a note by ID.
// POST /api/v1/resource/note/delete
func (c *Client) NoteDelete(noteID string) (*NoteDeleteResponse, error) {
	return doPost[NoteDeleteResponse](c, "/api/v1/resource/note/delete", NoteDeleteRequest{ID: noteID})
}

// NoteTaskRequest is the request body for querying task progress.
type NoteTaskRequest struct {
	TaskID string `json:"task_id"`
}

// NoteTaskResponse is the response from the task progress endpoint.
type NoteTaskResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// NoteTask queries the progress of a note-save task.
// POST /api/v1/resource/note/task/progress
func (c *Client) NoteTask(taskID string) (*NoteTaskResponse, error) {
	return doPost[NoteTaskResponse](c, "/api/v1/resource/note/task/progress", NoteTaskRequest{TaskID: taskID})
}

// ---------------------------------------------------------------------------
// Knowledge Base API
// ---------------------------------------------------------------------------

// KBListResponse is the response from the knowledge base list endpoint.
type KBListResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// KBList fetches all knowledge bases.
// GET /api/v1/resource/knowledge/list
func (c *Client) KBList() (*KBListResponse, error) {
	return doGet[KBListResponse](c, "/api/v1/resource/knowledge/list", nil)
}

// KBCreateRequest is the request body for creating a knowledge base.
type KBCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// KBCreateResponse is the response from the knowledge base create endpoint.
type KBCreateResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// KBCreate creates a new knowledge base.
// POST /api/v1/resource/knowledge/create
func (c *Client) KBCreate(req KBCreateRequest) (*KBCreateResponse, error) {
	return doPost[KBCreateResponse](c, "/api/v1/resource/knowledge/create", req)
}

// KBNotesParams holds parameters for listing notes in a knowledge base.
type KBNotesParams struct {
	TopicID string
	Limit   int
}

// KBNotesResponse is the response from the knowledge base notes endpoint.
type KBNotesResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// KBNotes fetches notes in a knowledge base.
// GET /api/v1/resource/knowledge/notes
func (c *Client) KBNotes(params KBNotesParams) (*KBNotesResponse, error) {
	q := url.Values{"topic_id": {params.TopicID}}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	return doGet[KBNotesResponse](c, "/api/v1/resource/knowledge/notes", q)
}

// KBNotesBatchAddRequest is the request body for batch-adding notes to a KB.
type KBNotesBatchAddRequest struct {
	TopicID string   `json:"topic_id"`
	NoteIDs []string `json:"note_ids"`
}

// KBNotesBatchAddResponse is the response from the batch-add endpoint.
type KBNotesBatchAddResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// KBNotesAdd adds notes to a knowledge base.
// POST /api/v1/resource/knowledge/note/batch-add
func (c *Client) KBNotesAdd(topicID string, noteIDs []string) (*KBNotesBatchAddResponse, error) {
	req := KBNotesBatchAddRequest{TopicID: topicID, NoteIDs: noteIDs}
	return doPost[KBNotesBatchAddResponse](c, "/api/v1/resource/knowledge/note/batch-add", req)
}

// KBNotesRemoveRequest is the request body for removing notes from a KB.
type KBNotesRemoveRequest struct {
	TopicID string   `json:"topic_id"`
	NoteIDs []string `json:"note_ids"`
}

// KBNotesRemoveResponse is the response from the remove endpoint.
type KBNotesRemoveResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// KBNotesRemove removes notes from a knowledge base.
// POST /api/v1/resource/knowledge/note/remove
func (c *Client) KBNotesRemove(topicID string, noteIDs []string) (*KBNotesRemoveResponse, error) {
	req := KBNotesRemoveRequest{TopicID: topicID, NoteIDs: noteIDs}
	return doPost[KBNotesRemoveResponse](c, "/api/v1/resource/knowledge/note/remove", req)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
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
