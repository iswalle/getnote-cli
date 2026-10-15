package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func positiveDecimalID(value string) (string, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strings.Trim(value, "0123456789") != "" {
		return "", fmt.Errorf("ID must be a positive signed-64-bit decimal string")
	}
	return strconv.FormatInt(id, 10), nil
}

type ReferenceLookupOptions struct {
	Source, ResourceType, Query, Author, MatchMode, Cursor string
	PageSize                                               int
}

func (o ReferenceLookupOptions) values() (url.Values, map[string]string, error) {
	filters := map[string]string{"source": strings.TrimSpace(o.Source), "resource_type": strings.TrimSpace(o.ResourceType), "query": strings.TrimSpace(o.Query), "author": strings.TrimSpace(o.Author), "match_mode": strings.TrimSpace(o.MatchMode)}
	if filters["match_mode"] == "" {
		filters["match_mode"] = "exact"
	}
	if filters["source"] != "dedao" || filters["resource_type"] != "course" || (filters["query"] == "" && filters["author"] == "") {
		return nil, nil, fmt.Errorf("reference lookup requires source=dedao, resource_type=course and query or author")
	}
	if filters["match_mode"] != "exact" && filters["match_mode"] != "prefix" {
		return nil, nil, fmt.Errorf("match_mode must be exact or prefix")
	}
	if utf8.RuneCountInString(filters["query"]) > 200 || utf8.RuneCountInString(filters["author"]) > 200 || len(o.Cursor) > 4096 {
		return nil, nil, fmt.Errorf("reference lookup exceeds input budget")
	}
	if o.PageSize == 0 {
		o.PageSize = 20
	}
	if o.PageSize < 1 || o.PageSize > 100 {
		return nil, nil, fmt.Errorf("page_size must be 1..100")
	}
	q := url.Values{}
	for k, v := range filters {
		if v != "" {
			q.Set(k, v)
		} else {
			delete(filters, k)
		}
	}
	q.Set("page_size", strconv.Itoa(o.PageSize))
	if o.Cursor != "" {
		q.Set("cursor", o.Cursor)
	}
	return q, filters, nil
}

type ReferenceResource struct {
	Source        string `json:"source"`
	ResourceType  string `json:"resource_type"`
	CourseType    string `json:"course_type"`
	CourseID      string `json:"course_id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	URL           string `json:"url"`
	HasNotes      bool   `json:"has_notes"`
	MetadataState string `json:"metadata_state"`
	rawFields     map[string]json.RawMessage
}

func (v *ReferenceResource) UnmarshalJSON(raw []byte) error {
	type plain ReferenceResource
	var p plain
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	*v = ReferenceResource(p)
	return json.Unmarshal(raw, &v.rawFields)
}
func (v ReferenceResource) MarshalJSON() ([]byte, error) {
	type plain ReferenceResource
	return preserveWireFields(plain(v), v.rawFields)
}

type ReferenceListData struct {
	DiscoveryListFields
	Resources []ReferenceResource `json:"resources"`
	Partial   bool                `json:"partial"`
}

func (v ReferenceListData) MarshalJSON() ([]byte, error) {
	type plain ReferenceListData
	return marshalDiscoveryData(plain(v), v.AppliedFilters)
}

type ReferenceListResponse struct {
	Success   bool              `json:"success"`
	Data      ReferenceListData `json:"data"`
	Meta      *ResponseMeta     `json:"meta,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func (c *Client) ReferenceList(o ReferenceLookupOptions) (*ReferenceListResponse, error) {
	q, filters, err := o.values()
	if err != nil {
		return nil, err
	}
	r, err := doGet[ReferenceListResponse](c, "/open/api/v1/resource/reference/list", q)
	if err != nil {
		return nil, err
	}
	if err = requireCapabilities(r.Meta, "reference.resources.v1", "pagination.v1"); err != nil {
		return nil, err
	}
	if err = verifyApplied(r.Data.AppliedFilters, filters); err != nil {
		return nil, err
	}
	size, _ := strconv.Atoi(q.Get("page_size"))
	if err = verifyPagination(r.Data.Pagination, size, len(r.Data.Resources), "cursor", 0); err != nil {
		return nil, err
	}
	if r.Data.Pagination.TotalCount != nil {
		return nil, fmt.Errorf("course reference total must be unknown")
	}
	for _, v := range r.Data.Resources {
		if _, err = positiveDecimalID(v.CourseID); err != nil || v.CourseType == "" {
			return nil, fmt.Errorf("missing course identity")
		}
	}
	return r, nil
}

// A bounded lookup can return an empty page after checking ownership. The
// advancing continuation, rather than the number of rows, determines progress.
func (c *Client) AllReferences(o ReferenceLookupOptions) (*ReferenceListResponse, error) {
	rows := []ReferenceResource{}
	seenCursors := map[string]bool{}
	seenIDs := map[string]bool{}
	var out *ReferenceListResponse
	for page := 1; page <= 1000; page++ {
		if seenCursors[o.Cursor] {
			return nil, fmt.Errorf("reference traversal incomplete after %d pages: repeated cursor", page-1)
		}
		seenCursors[o.Cursor] = true
		r, err := c.ReferenceList(o)
		if err != nil {
			return nil, fmt.Errorf("reference traversal incomplete after %d pages: %w", page-1, err)
		}
		pageRows := r.Data.Resources
		if out == nil {
			out = r
			out.Data.Resources = nil
		} else {
			out.Data.Partial = out.Data.Partial || r.Data.Partial
			out.Data.Warnings = append(out.Data.Warnings, r.Data.Warnings...)
			out.Meta = r.Meta
			out.RequestID = r.RequestID
		}
		for _, v := range pageRows {
			id := v.CourseType + ":" + v.CourseID
			if !seenIDs[id] {
				seenIDs[id] = true
				rows = append(rows, v)
			}
		}
		p := r.Data.Pagination
		if !p.HasMore {
			out.Data.Resources = rows
			out.Data.Pagination = nil
			out.Data.TotalScope = "returned_in_traversal"
			out.Data.Traversal = &Traversal{Completed: true, Pages: page, ReturnedCount: len(rows), PaginationVerified: true}
			return out, nil
		}
		o.Cursor = *p.NextCursor
	}
	return nil, fmt.Errorf("reference traversal incomplete: exceeded 1000-page safety budget")
}

type BatchDetailOptions struct {
	IDs           []string `json:"ids"`
	Fields        []string `json:"fields,omitempty"`
	MaxTotalChars int      `json:"max_total_chars,omitempty"`
}

func (o BatchDetailOptions) normalized() (BatchDetailOptions, error) {
	if len(o.IDs) == 0 || len(o.IDs) > 200 {
		return o, fmt.Errorf("batch requires 1..20 unique IDs, at most 200 raw entries")
	}
	out := o
	out.IDs = nil
	seen := map[string]bool{}
	for _, raw := range o.IDs {
		id, err := positiveDecimalID(raw)
		if err != nil {
			return out, err
		}
		if !seen[id] {
			seen[id] = true
			out.IDs = append(out.IDs, id)
		}
	}
	if len(out.IDs) > 20 {
		return out, fmt.Errorf("batch supports at most 20 unique IDs")
	}
	if o.MaxTotalChars < 0 || o.MaxTotalChars > 64000 {
		return out, fmt.Errorf("max_total_chars must be 1..64000 or omitted")
	}
	for _, field := range o.Fields {
		if field != "content" && field != "ref_content" && field != "reference" && field != "original" {
			return out, fmt.Errorf("unsupported fields value: %s", field)
		}
	}
	return out, nil
}

type BatchDetailItem struct {
	NoteID string          `json:"note_id"`
	Status string          `json:"status"`
	Note   json.RawMessage `json:"note,omitempty"`
}
type BatchDetailResponse struct {
	Success   bool          `json:"success"`
	Meta      *ResponseMeta `json:"meta,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
	Data      struct {
		Items         []BatchDetailItem `json:"items"`
		PendingIDs    []string          `json:"pending_ids"`
		TotalChars    int               `json:"total_chars"`
		MaxTotalChars int               `json:"max_total_chars"`
		MaxTotalBytes int               `json:"max_total_bytes"`
		Partial       bool              `json:"partial"`
	} `json:"data"`
}

func (c *Client) NotesBatch(o BatchDetailOptions) (*BatchDetailResponse, error) {
	o, err := o.normalized()
	if err != nil {
		return nil, err
	}
	r, err := doPost[BatchDetailResponse](c, "/open/api/v1/resource/note/batch-detail", o)
	if err != nil {
		return nil, err
	}
	if err = requireCapabilities(r.Meta, "note.batch_detail.v1"); err != nil {
		return nil, err
	}
	if len(r.Data.Items) != len(o.IDs) {
		return nil, fmt.Errorf("batch response lost item identities")
	}
	for i, item := range r.Data.Items {
		if item.NoteID != o.IDs[i] || item.Status == "" {
			return nil, fmt.Errorf("batch response changed item identities or omitted status")
		}
	}
	return r, nil
}

type SearchOptions struct {
	ResponseMode    string `json:"response_mode,omitempty"`
	MaxSnippetChars *int   `json:"max_snippet_chars,omitempty"`
}

func (c *Client) SearchWithOptions(topicID, query string, topK int, o SearchOptions) (*NoteSearchResponse, error) {
	if o.ResponseMode == "" && o.MaxSnippetChars == nil {
		if topicID != "" {
			return c.KBSearch(topicID, query, topK)
		}
		return c.NoteSearch(query, topK)
	}
	if o.ResponseMode != "snippet" {
		return nil, fmt.Errorf("response_mode must be snippet when using snippet options")
	}
	limit := 400
	if o.MaxSnippetChars != nil {
		limit = *o.MaxSnippetChars
	}
	if limit < 100 || limit > 1000 {
		return nil, fmt.Errorf("max_snippet_chars must be 100..1000")
	}
	request := struct {
		SearchOptions
		Query   string `json:"query"`
		TopK    int    `json:"top_k,omitempty"`
		TopicID string `json:"topic_id,omitempty"`
	}{o, query, topK, topicID}
	path := "/open/api/v1/resource/recall"
	if topicID != "" {
		path += "/knowledge"
	}
	r, err := doPost[NoteSearchResponse](c, path, request)
	if err != nil {
		return nil, err
	}
	if err = requireCapabilities(r.Meta, "search.snippet.v1"); err != nil {
		return nil, err
	}
	if r.Data.ResponseMode != "snippet" || r.Data.MaxSnippetChars != limit {
		return nil, fmt.Errorf("server ignored snippet options")
	}
	c.decorateSearch(r)
	return r, nil
}
func (v *RecallResult) UnmarshalJSON(raw []byte) error {
	type plain RecallResult
	var p plain
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	*v = RecallResult(p)
	return json.Unmarshal(raw, &v.rawFields)
}
func (v RecallResult) MarshalJSON() ([]byte, error) {
	type plain RecallResult
	return preserveWireFields(plain(v), v.rawFields)
}
