package client

import (
	"fmt"
	"net/url"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

type NoteSectionOptions struct {
	Offset, Limit    *int
	Keyword, Version string
}
type NoteSectionData struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Total      int    `json:"total"`
	Version    string `json:"version"`
	OffsetUnit string `json:"offset_unit"`
}
type NoteSectionResponse struct {
	Success   bool            `json:"success"`
	Data      NoteSectionData `json:"data"`
	Meta      *ResponseMeta   `json:"meta,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

func (c *Client) NoteOutline(id string) (*NoteSectionResponse, error) {
	return c.notePage(id, "outline", NoteSectionOptions{})
}
func (c *Client) NoteSection(id string, options NoteSectionOptions) (*NoteSectionResponse, error) {
	return c.notePage(id, "section", options)
}
func (c *Client) notePage(id, operation string, o NoteSectionOptions) (*NoteSectionResponse, error) {
	id, err := positiveDecimalID(id)
	if err != nil {
		return nil, err
	}
	q := url.Values{"id": {id}}
	if o.Offset != nil {
		if *o.Offset < 0 || *o.Offset > 3_000_000 {
			return nil, fmt.Errorf("offset must be 0..3000000 UTF-16 units")
		}
		q.Set("offset", strconv.Itoa(*o.Offset))
	}
	if o.Limit != nil {
		if *o.Limit < 1 || *o.Limit > 30000 {
			return nil, fmt.Errorf("limit must be 1..30000 UTF-16 units")
		}
		q.Set("limit", strconv.Itoa(*o.Limit))
	}
	if !utf8.ValidString(o.Keyword) || len(utf16.Encode([]rune(o.Keyword))) > 1000 || !utf8.ValidString(o.Version) || len(o.Version) > 80 {
		return nil, fmt.Errorf("invalid keyword or version")
	}
	if o.Keyword != "" {
		q.Set("keyword", o.Keyword)
	}
	if o.Version != "" {
		q.Set("version", o.Version)
	}
	r, err := doGet[NoteSectionResponse](c, "/open/api/v1/resource/note/"+operation, q)
	if err != nil {
		return nil, err
	}
	if err = requireCapabilities(r.Meta, "note.progressive_read.v1"); err != nil {
		return nil, err
	}
	if r.Data.ID != id || r.Data.OffsetUnit != "utf16" || r.Data.Version == "" || r.Data.Text == "" || r.Data.Total < 0 || r.Data.Total > 3_000_000 || len(utf16.Encode([]rune(r.Data.Text))) > 64000 || o.Version != "" && r.Data.Version != o.Version {
		return nil, fmt.Errorf("invalid progressive read response; stop and refresh outline, never fall back to full content automatically")
	}
	return r, nil
}
