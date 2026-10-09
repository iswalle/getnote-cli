package client

import (
	"fmt"
	"golang.org/x/text/unicode/norm"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DiscoveryOptions struct {
	View, NoteType, Source, ReferenceType, ReferenceID, ArticleURL, TopicID          string
	CreatedFrom, CreatedTo, EditedFrom, EditedTo, RecordUpdatedFrom, RecordUpdatedTo string
	Cursor                                                                           string
	Page, PageSize                                                                   int
}
type KnowledgeLookupOptions struct {
	Scope, Query, Author, BookID string
	Page, PageSize               int
}

func (o DiscoveryOptions) Active() bool {
	return (o.Cursor != "" && strings.Trim(o.Cursor, "0123456789") != "") || o.View != "" || o.NoteType != "" || o.Source != "" || o.ReferenceType != "" || o.ReferenceID != "" || o.ArticleURL != "" || o.TopicID != "" || o.CreatedFrom != "" || o.CreatedTo != "" || o.EditedFrom != "" || o.EditedTo != "" || o.RecordUpdatedFrom != "" || o.RecordUpdatedTo != "" || o.PageSize != 0 || o.Page != 0
}

func (o DiscoveryOptions) values() (url.Values, map[string]string, error) {
	q := url.Values{}
	filters := map[string]string{}
	if o.View != "" && o.View != "summary" && o.View != "full" {
		return nil, nil, fmt.Errorf("view must be summary or full")
	}
	if o.PageSize == 0 {
		o.PageSize = 20
	}
	if o.PageSize < 1 || o.PageSize > 100 {
		return nil, nil, fmt.Errorf("page_size must be 1..100")
	}
	if o.Page < 0 || (o.Page > 0 && (o.Cursor != "" || o.TopicID == "")) {
		return nil, nil, fmt.Errorf("page requires a topic and cannot be combined with cursor")
	}
	for k, v := range map[string]string{"source": o.Source, "note_type": o.NoteType, "reference_resource_type": o.ReferenceType, "reference_resource_id": o.ReferenceID, "topic_id": o.TopicID} {
		if v = strings.TrimSpace(v); v != "" {
			filters[k] = v
		}
	}
	if id := filters["reference_resource_id"]; id != "" {
		if filters["source"] == "" || filters["reference_resource_type"] == "" {
			return nil, nil, fmt.Errorf("reference ID requires source and reference type")
		}
		if n, err := strconv.ParseInt(id, 10, 64); err != nil || n <= 0 || strings.Trim(id, "0123456789") != "" {
			return nil, nil, fmt.Errorf("invalid reference ID")
		}
	}
	if o.ArticleURL != "" {
		u, err := url.Parse(o.ArticleURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, nil, fmt.Errorf("invalid article URL")
		}
		filters["reference_article_url"] = o.ArticleURL
	}
	for k, v := range map[string]string{"created_from": o.CreatedFrom, "created_to": o.CreatedTo, "edited_from": o.EditedFrom, "edited_to": o.EditedTo, "record_updated_from": o.RecordUpdatedFrom, "record_updated_to": o.RecordUpdatedTo} {
		if v != "" {
			tm, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return nil, nil, fmt.Errorf("%s must be RFC3339 with timezone", k)
			}
			filters[k] = tm.UTC().Format(time.RFC3339Nano)
		}
	}
	for _, prefix := range []string{"created", "edited", "record_updated"} {
		from, to := filters[prefix+"_from"], filters[prefix+"_to"]
		if from != "" && to != "" {
			a, _ := time.Parse(time.RFC3339Nano, from)
			b, _ := time.Parse(time.RFC3339Nano, to)
			if !a.Before(b) {
				return nil, nil, fmt.Errorf("%s interval must have from < to", prefix)
			}
		}
	}
	for k, v := range filters {
		q.Set(k, v)
	}
	q.Set("page_size", strconv.Itoa(o.PageSize))
	if o.Cursor != "" {
		q.Set("cursor", o.Cursor)
	}
	if o.Page > 0 {
		q.Set("page", strconv.Itoa(o.Page))
	}
	if o.View != "" {
		q.Set("view", o.View)
	}
	return q, filters, nil
}

func requireCapabilities(meta *ResponseMeta, caps ...string) error {
	for _, capability := range caps {
		found := false
		if meta != nil {
			for _, advertised := range meta.Capabilities {
				if advertised == capability {
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("server does not support %s; update the server or remove the new options (unfiltered results were not accepted)", capability)
		}
	}
	return nil
}
func verifyApplied(actual, expected map[string]string) error {
	if actual == nil {
		return fmt.Errorf("server did not return applied_filters")
	}
	for k, v := range expected {
		if actual[k] != v {
			return fmt.Errorf("server did not apply %s=%q", k, v)
		}
	}
	return nil
}
func verifyPagination(p *Pagination, size, count int, mode string, page int) error {
	if p == nil || p.Mode != mode || p.PageSize != size || p.ReturnedCount != count || count > size {
		return fmt.Errorf("invalid or unsupported pagination response")
	}
	if p.TotalCount != nil && *p.TotalCount < int64(count) {
		return fmt.Errorf("invalid pagination total_count")
	}
	if mode == "page" && (p.Page == nil || *p.Page != page) {
		return fmt.Errorf("server returned a different page")
	}
	if p.HasMore {
		if mode == "cursor" && (p.NextCursor == nil || *p.NextCursor == "" || p.NextPage != nil) {
			return fmt.Errorf("missing next cursor")
		}
		if mode == "page" && (p.NextPage == nil || *p.NextPage != page+1 || p.NextCursor != nil) {
			return fmt.Errorf("invalid next page")
		}
	} else if p.NextCursor != nil || p.NextPage != nil {
		return fmt.Errorf("finished page has a continuation")
	}
	return nil
}
func validateDiscoveryPage(meta *ResponseMeta, data DiscoveryListFields, q url.Values, filters map[string]string, count int, lookup bool) error {
	caps := []string{"pagination.v1"}
	if lookup {
		caps = append(caps, "knowledge.lookup.v1")
	} else if len(filters) > 0 {
		caps = append(caps, "note.filters.v1")
	}
	if q.Get("view") == "summary" {
		caps = append(caps, "note.summary.v1", "note.reference.v1")
	}
	if err := requireCapabilities(meta, caps...); err != nil {
		return err
	}
	if err := verifyApplied(data.AppliedFilters, filters); err != nil {
		return err
	}
	size, _ := strconv.Atoi(q.Get("page_size"))
	page, _ := strconv.Atoi(q.Get("page"))
	mode := "cursor"
	if page > 0 {
		mode = "page"
	}
	return verifyPagination(data.Pagination, size, count, mode, page)
}

func (c *Client) NoteListWithOptions(o DiscoveryOptions) (*NoteListResponse, error) {
	q, filters, err := o.values()
	if err != nil {
		return nil, err
	}
	resp, err := doGet[NoteListResponse](c, "/open/api/v1/resource/note/list", q)
	if err != nil {
		return nil, err
	}
	if err = validateDiscoveryPage(resp.Meta, resp.Data.DiscoveryListFields, q, filters, len(resp.Data.Notes), false); err != nil {
		return nil, err
	}
	resp.Data.HasMore = resp.Data.Pagination.HasMore
	if resp.Data.Pagination.NextCursor != nil {
		resp.Data.Cursor = *resp.Data.Pagination.NextCursor
	}
	for i := range resp.Data.Notes {
		if resp.Data.Notes[i].NoteURL == "" {
			resp.Data.Notes[i].NoteURL = c.NoteURL(noteIdentity(resp.Data.Notes[i]))
		}
	}
	return resp, nil
}
func (c *Client) KBNotesWithOptions(topicID string, o DiscoveryOptions) (*KBNotesResponse, error) {
	if topicID == "" || o.TopicID != "" && o.TopicID != topicID {
		return nil, fmt.Errorf("conflicting or missing topic ID")
	}
	o.TopicID = topicID
	if o.Page == 0 {
		o.Page = 1
	}
	q, filters, err := o.values()
	if err != nil {
		return nil, err
	}
	resp, err := doGet[KBNotesResponse](c, "/open/api/v1/resource/knowledge/notes", q)
	if err != nil {
		return nil, err
	}
	if err = validateDiscoveryPage(resp.Meta, resp.Data.DiscoveryListFields, q, filters, len(resp.Data.Notes), false); err != nil {
		return nil, err
	}
	resp.Data.HasMore = resp.Data.Pagination.HasMore
	return resp, nil
}
func (c *Client) NoteGetWithOptions(id, view string) (*NoteGetResponse, error) {
	if view == "" {
		return c.NoteGet(id)
	}
	if view != "full" && view != "summary" {
		return nil, fmt.Errorf("view must be summary or full")
	}
	resp, err := doGet[NoteGetResponse](c, "/open/api/v1/resource/note/detail", url.Values{"id": {id}, "view": {view}})
	if err != nil {
		return nil, err
	}
	caps := []string{"note.reference.v1"}
	if view == "summary" {
		caps = append(caps, "note.summary.v1")
	}
	if err = requireCapabilities(resp.Meta, caps...); err != nil {
		return nil, err
	}
	if resp.Data.Note.NoteURL == "" {
		resp.Data.Note.NoteURL = c.NoteURL(noteIdentity(resp.Data.Note))
	}
	return resp, nil
}
func normalizeLookup(value string) string {
	value = strings.TrimSpace(norm.NFKC.String(value))
	for _, pair := range [][2]string{{"《", "》"}, {"“", "”"}, {"\"", "\""}, {"'", "'"}} {
		if strings.HasPrefix(value, pair[0]) && strings.HasSuffix(value, pair[1]) && len(value) >= len(pair[0])+len(pair[1]) {
			value = strings.TrimSpace(value[len(pair[0]) : len(value)-len(pair[1])])
			break
		}
	}
	return strings.ToLower(value)
}
func (c *Client) KBListWithOptions(o KnowledgeLookupOptions, subscribed bool) (*KBListResponse, error) {
	q := knowledgeListValues(o.Page, o.Scope)
	scope := q.Get("scope")
	if o.Page < 0 {
		return nil, fmt.Errorf("page must be positive")
	}
	if scope != "BOOKSPACE" && (o.Author != "" || o.BookID != "") {
		return nil, fmt.Errorf("author and book ID require BOOKSPACE")
	}
	if o.PageSize == 0 {
		o.PageSize = 20
	}
	if o.PageSize < 1 || o.PageSize > 100 {
		return nil, fmt.Errorf("page_size must be 1..100")
	}
	filters := map[string]string{"scope": scope}
	for k, v := range map[string]string{"query": normalizeLookup(o.Query), "author": normalizeLookup(o.Author), "book_id": strings.TrimSpace(o.BookID)} {
		if v != "" {
			filters[k] = v
			q.Set(k, v)
		}
	}
	q.Set("page_size", strconv.Itoa(o.PageSize))
	path := "/open/api/v1/resource/knowledge/list"
	if subscribed {
		path = "/open/api/v1/resource/knowledge/subscribe/list"
	}
	resp, err := doGet[KBListResponse](c, path, q)
	if err != nil {
		return nil, err
	}
	if err = validateDiscoveryPage(resp.Meta, resp.Data.DiscoveryListFields, q, filters, len(resp.Data.Topics), true); err != nil {
		return nil, err
	}
	resp.Data.HasMore = resp.Data.Pagination.HasMore
	return resp, nil
}
