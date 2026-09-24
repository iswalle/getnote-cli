package client

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type ResourceResponse struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id,omitempty"`
}

func (c *Client) NoteMarks(id string) (*ResourceResponse, error) {
	return doGet[ResourceResponse](c, "/open/api/v1/resource/note/marks", url.Values{"note_id": {id}})
}
func (c *Client) Sprouts(month string) (*ResourceResponse, error) {
	return c.SproutsPage(month, "", 20)
}
func (c *Client) SproutsPage(month, sinceID string, limit int) (*ResourceResponse, error) {
	query := url.Values{"month": {month}, "limit": {strconv.Itoa(limit)}}
	if sinceID != "" {
		query.Set("since_id", sinceID)
	}
	return doGet[ResourceResponse](c, "/open/api/v1/resource/note/sprouts", query)
}
func (c *Client) Sprout(id string) (*ResourceResponse, error) {
	return doGet[ResourceResponse](c, "/open/api/v1/resource/note/sprout", url.Values{"id": {id}})
}
func (c *Client) FileCapabilities() (*ResourceResponse, error) {
	return doGet[ResourceResponse](c, "/open/api/v1/resource/knowledge/file/capabilities", nil)
}
func (c *Client) FileToken(ext string) (*ResourceResponse, error) {
	return doGet[ResourceResponse](c, "/open/api/v1/resource/knowledge/file/upload_token", url.Values{"mime_type": {ext}})
}
func (c *Client) FileAdd(req any) (*ResourceResponse, error) {
	return doPost[ResourceResponse](c, "/open/api/v1/resource/knowledge/file/upload", req)
}
