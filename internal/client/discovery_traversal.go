package client

import "fmt"

type discoveryPage[T any] struct {
	items  []T
	fields DiscoveryListFields
	meta   *ResponseMeta
}

func walkDiscovery[T any](cursor string, page int, strict bool, fetch func(string, int) (discoveryPage[T], error), identity func(T) string) ([]T, DiscoveryListFields, *ResponseMeta, error) {
	all := make([]T, 0)
	seenIDs := map[string]bool{}
	continuations := map[string]bool{}
	pages := 0
	fields := DiscoveryListFields{}
	var meta *ResponseMeta
	fail := func(err error) ([]T, DiscoveryListFields, *ResponseMeta, error) {
		return nil, DiscoveryListFields{}, nil, fmt.Errorf("遍历未完成（已读取 %d 页、%d 条唯一记录）: %w", pages, len(all), err)
	}
	for {
		key := fmt.Sprintf("%d:%s", page, cursor)
		if continuations[key] {
			return fail(fmt.Errorf("repeated continuation"))
		}
		continuations[key] = true
		result, err := fetch(cursor, page)
		if err != nil {
			return fail(err)
		}
		pages++
		p := result.fields.Pagination
		if p == nil {
			return fail(fmt.Errorf("pagination unavailable"))
		}
		fields.AppliedFilters = result.fields.AppliedFilters
		fields.Warnings = append(fields.Warnings, result.fields.Warnings...)
		fields.Partial = fields.Partial || result.fields.Partial || len(result.fields.Warnings) > 0
		if len(fields.Coverage) == 0 && len(result.fields.Coverage) > 0 {
			fields.Coverage = append([]byte(nil), result.fields.Coverage...)
		}
		meta = result.meta
		added := 0
		for _, item := range result.items {
			id := identity(item)
			if id == "" {
				return fail(fmt.Errorf("missing record identity"))
			}
			if !seenIDs[id] {
				seenIDs[id] = true
				all = append(all, item)
				added++
			}
		}
		if pages > 1 && added == 0 && (strict || p.HasMore || len(result.items) > 0) {
			return fail(fmt.Errorf("continuation page made no progress"))
		}
		if !p.HasMore {
			fields.Traversal = &Traversal{Completed: !fields.Partial, Pages: pages, ReturnedCount: len(all), PaginationVerified: strict}
			fields.TotalScope = "returned_in_traversal"
			return all, fields, meta, nil
		}
		if added == 0 {
			return fail(fmt.Errorf("intermediate page made no progress"))
		}
		switch p.Mode {
		case "cursor":
			if p.NextCursor == nil || *p.NextCursor == "" {
				return fail(fmt.Errorf("missing next cursor"))
			}
			cursor = *p.NextCursor
		case "page":
			if p.NextPage == nil || *p.NextPage <= page {
				return fail(fmt.Errorf("invalid next page"))
			}
			page = *p.NextPage
		default:
			return fail(fmt.Errorf("unknown pagination mode"))
		}
	}
}

func (c *Client) AllNotes(o DiscoveryOptions) (*NoteListResponse, error) {
	items, fields, meta, err := walkDiscovery(o.Cursor, o.Page, true, func(cursor string, page int) (discoveryPage[Note], error) {
		next := o
		next.Cursor = cursor
		next.Page = page
		r, err := c.NoteListWithOptions(next)
		if err != nil {
			return discoveryPage[Note]{}, err
		}
		return discoveryPage[Note]{r.Data.Notes, r.Data.DiscoveryListFields, r.Meta}, nil
	}, noteIdentity)
	if err != nil {
		return nil, err
	}
	return &NoteListResponse{Success: true, Meta: meta, Data: NoteListData{DiscoveryListFields: fields, Notes: items, Total: legacyReturnedTotal(len(items))}}, nil
}
func (c *Client) AllKBNotes(topicID string, o DiscoveryOptions) (*KBNotesResponse, error) {
	if o.Page == 0 {
		o.Page = 1
	}
	var topic []byte
	items, fields, meta, err := walkDiscovery(o.Cursor, o.Page, true, func(cursor string, page int) (discoveryPage[KBNote], error) {
		next := o
		next.Cursor = cursor
		next.Page = page
		r, err := c.KBNotesWithOptions(topicID, next)
		if err != nil {
			return discoveryPage[KBNote]{}, err
		}
		topic = r.Data.Topic
		return discoveryPage[KBNote]{r.Data.Notes, r.Data.DiscoveryListFields, r.Meta}, nil
	}, func(n KBNote) string { return n.NoteID })
	if err != nil {
		return nil, err
	}
	return &KBNotesResponse{Success: true, Meta: meta, Data: KBNoteListData{DiscoveryListFields: fields, Notes: items, Total: legacyReturnedTotal(len(items)), Topic: topic}}, nil
}
func (c *Client) AllKnowledge(o KnowledgeLookupOptions, subscribed bool) (*KBListResponse, error) {
	if o.Page == 0 {
		o.Page = 1
	}
	items, fields, meta, err := walkDiscovery("", o.Page, true, func(_ string, page int) (discoveryPage[KBTopic], error) {
		next := o
		next.Page = page
		r, err := c.KBListWithOptions(next, subscribed)
		if err != nil {
			return discoveryPage[KBTopic]{}, err
		}
		return discoveryPage[KBTopic]{r.Data.Topics, r.Data.DiscoveryListFields, r.Meta}, nil
	}, func(n KBTopic) string {
		if n.TopicID != "" {
			return n.TopicID
		}
		return n.ID
	})
	if err != nil {
		return nil, err
	}
	return &KBListResponse{Success: true, Meta: meta, Data: KBListData{DiscoveryListFields: fields, Topics: items, Total: legacyReturnedTotal(len(items))}}, nil
}

// Legacy walks do not claim support for new pagination. They retain the old
// response semantics and still guard cycles, duplicate pages and HTTP errors.
func (c *Client) LegacyAllNotes(cursor string) (*NoteListResponse, error) {
	items, fields, meta, err := walkDiscovery(cursor, 0, false, func(cur string, _ int) (discoveryPage[Note], error) {
		r, err := c.NoteList(NoteListParams{Cursor: cur, Limit: 20})
		if err != nil {
			return discoveryPage[Note]{}, err
		}
		p := &Pagination{Mode: "cursor", HasMore: r.Data.HasMore, ReturnedCount: len(r.Data.Notes), PageSize: 20}
		if p.HasMore {
			next := r.Data.Cursor
			if next == "" {
				next = r.Data.NextCursor.String()
			}
			p.NextCursor = &next
		}
		return discoveryPage[Note]{r.Data.Notes, DiscoveryListFields{Pagination: p}, r.Meta}, nil
	}, noteIdentity)
	if err != nil {
		return nil, err
	}
	return &NoteListResponse{Success: true, Meta: meta, Data: NoteListData{DiscoveryListFields: fields, Notes: items, Total: legacyReturnedTotal(len(items))}}, nil
}
func (c *Client) LegacyAllKBNotes(topicID string) (*KBNotesResponse, error) {
	items, fields, meta, err := walkDiscovery("", 1, false, func(_ string, page int) (discoveryPage[KBNote], error) {
		r, err := c.KBNotes(KBNotesParams{TopicID: topicID, Page: page, Limit: 20})
		if err != nil {
			return discoveryPage[KBNote]{}, err
		}
		p := &Pagination{Mode: "page", Page: &page, HasMore: r.Data.HasMore, ReturnedCount: len(r.Data.Notes), PageSize: 20}
		if p.HasMore {
			next := page + 1
			p.NextPage = &next
		}
		return discoveryPage[KBNote]{r.Data.Notes, DiscoveryListFields{Pagination: p}, r.Meta}, nil
	}, func(n KBNote) string { return n.NoteID })
	if err != nil {
		return nil, err
	}
	return &KBNotesResponse{Success: true, Meta: meta, Data: KBNoteListData{DiscoveryListFields: fields, Notes: items, Total: legacyReturnedTotal(len(items))}}, nil
}
