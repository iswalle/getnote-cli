package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
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
	if parsed, err := time.Parse("2006-01", month); err != nil || parsed.Format("2006-01") != month {
		return nil, fmt.Errorf("month must be a valid YYYY-MM calendar month")
	}
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
	result, err := doPost[ResourceResponse](c, "/open/api/v1/resource/knowledge/file/upload", req)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &fields); err != nil {
		return nil, fmt.Errorf("invalid file upload result: %w", err)
	}
	for _, name := range []string{"id", "topic_id", "directory_id"} {
		raw, ok := fields[name]
		if !ok {
			continue
		}
		var id StringID
		if err := json.Unmarshal(raw, &id); err != nil {
			return nil, fmt.Errorf("invalid file upload %s: %w", name, err)
		}
		if value, err := strconv.ParseInt(id.String(), 10, 64); err != nil || value < 0 {
			return nil, fmt.Errorf("file upload accepted but %s is not a valid int64 ID; inspect kb directories before retrying", name)
		}
		key := name + "_str"
		if existing, ok := fields[key]; ok {
			var alias string
			if err := json.Unmarshal(existing, &alias); err != nil || alias != id.String() {
				return nil, fmt.Errorf("file upload %s does not match %s", key, name)
			}
		} else {
			fields[key], _ = json.Marshal(id.String())
		}
	}
	result.Data, err = json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return result, nil
}
