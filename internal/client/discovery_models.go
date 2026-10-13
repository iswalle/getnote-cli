package client

import "encoding/json"

type ResponseMeta struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Timestamp    int64    `json:"timestamp,omitempty"`
}

type Pagination struct {
	Mode          string  `json:"mode"`
	Page          *int    `json:"page"`
	PageSize      int     `json:"page_size"`
	ReturnedCount int     `json:"returned_count"`
	TotalCount    *int64  `json:"total_count"`
	HasMore       bool    `json:"has_more"`
	NextPage      *int    `json:"next_page"`
	NextCursor    *string `json:"next_cursor"`
}

type DiscoveryListFields struct {
	TotalScope string      `json:"total_scope,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	// Filters include string selectors and typed flags such as exclude_mine.
	AppliedFilters map[string]json.RawMessage `json:"applied_filters,omitempty"`
	Warnings       []json.RawMessage          `json:"warnings,omitempty"`
	Traversal      *Traversal                 `json:"traversal,omitempty"`
}

// nil means a legacy response omitted the field; a non-nil empty map is the
// server's affirmative statement that no filters were applied.
func marshalDiscoveryData(value any, filters map[string]json.RawMessage) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil || filters == nil {
		return encoded, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	fields["applied_filters"], err = json.Marshal(filters)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

func (d NoteListData) MarshalJSON() ([]byte, error) {
	type plain NoteListData
	return marshalDiscoveryData(plain(d), d.AppliedFilters)
}

func (d KBNoteListData) MarshalJSON() ([]byte, error) {
	type plain KBNoteListData
	return marshalDiscoveryData(plain(d), d.AppliedFilters)
}

func (d KBListData) MarshalJSON() ([]byte, error) {
	type plain KBListData
	return marshalDiscoveryData(plain(d), d.AppliedFilters)
}

// Traversal describes this walk; it never claims snapshot or global totals.
type Traversal struct {
	TotalCount         *int64 `json:"total_count"`
	PaginationVerified bool   `json:"pagination_verified"`
	Completed          bool   `json:"completed"`
	Pages              int    `json:"pages"`
	ReturnedCount      int    `json:"returned_count"`
}

type NoteReference struct {
	Source          string             `json:"source,omitempty"`
	ResourceType    string             `json:"resource_type,omitempty"`
	PType           *int64             `json:"ptype,omitempty"`
	ResourceID      string             `json:"resource_id,omitempty"`
	Title           string             `json:"title,omitempty"`
	Author          string             `json:"author,omitempty"`
	URL             string             `json:"url,omitempty"`
	OriginalNoteID  string             `json:"original_note_id,omitempty"`
	IsSourcePrivate *bool              `json:"is_source_private,omitempty"`
	BookID          string             `json:"book_id,omitempty"`
	Article         *ReferenceArticle  `json:"article,omitempty"`
	Location        *ReferenceLocation `json:"location,omitempty"`
}
type ReferenceArticle struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}
type ReferenceLocation struct {
	ChapterID   string `json:"chapter_id,omitempty"`
	ChapterName string `json:"chapter_name,omitempty"`
	Page        *int64 `json:"page,omitempty"`
	StartOffset *int64 `json:"start_offset,omitempty"`
	EndOffset   *int64 `json:"end_offset,omitempty"`
}
type NoteTimestamps struct {
	CreatedAt       string `json:"created_at,omitempty"`
	EditedAt        string `json:"edited_at,omitempty"`
	RecordUpdatedAt string `json:"record_updated_at,omitempty"`
}
type NotePreview struct {
	Text      string `json:"text"`
	Source    string `json:"source"`
	Truncated bool   `json:"truncated"`
}
type NoteContentState struct {
	Content    string `json:"content"`
	RefContent string `json:"ref_content"`
}
type NoteMetadataState struct {
	Reference string `json:"reference"`
	Topics    string `json:"topics"`
}
type DiscoveryNoteFields struct {
	DisplayTitle  string             `json:"display_title,omitempty"`
	Reference     *NoteReference     `json:"reference,omitempty"`
	Timestamps    *NoteTimestamps    `json:"timestamps,omitempty"`
	Preview       *NotePreview       `json:"preview,omitempty"`
	ContentState  *NoteContentState  `json:"content_state,omitempty"`
	MetadataState *NoteMetadataState `json:"metadata_state,omitempty"`
}

func (n Note) DisplayName() string {
	if n.DisplayTitle != "" {
		return n.DisplayTitle
	}
	return n.Title
}
func (n KBNote) DisplayName() string {
	if n.DisplayTitle != "" {
		return n.DisplayTitle
	}
	return n.Title
}

func (n *Note) UnmarshalJSON(data []byte) error {
	type plain Note
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*n = Note(value)
	return json.Unmarshal(data, &n.rawFields)
}
func (n Note) MarshalJSON() ([]byte, error) {
	type plain Note
	return preserveWireFields(plain(n), n.rawFields)
}
func (n *KBNote) UnmarshalJSON(data []byte) error {
	type plain KBNote
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*n = KBNote(value)
	return json.Unmarshal(data, &n.rawFields)
}
func (n KBNote) MarshalJSON() ([]byte, error) {
	type plain KBNote
	return preserveWireFields(plain(n), n.rawFields)
}

// Keep unknown server fields and, crucially, absent summary fields absent.
// Typed known fields still receive ID normalization and caller modifications.
func preserveWireFields(value any, original map[string]json.RawMessage) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil || original == nil {
		return encoded, err
	}
	var known map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &known); err != nil {
		return nil, err
	}
	merged := make(map[string]json.RawMessage, len(original))
	for k, v := range original {
		merged[k] = v
	}
	for k, v := range known {
		if _, present := original[k]; present || k == "note_url" {
			merged[k] = v
		}
	}
	return json.Marshal(merged)
}

func legacyReturnedTotal(count int) *int { return &count }

// OmitContent supports the existing local --no-content operation without
// representing unloaded content as an empty user-written string.
func (n *KBNote) OmitContent() {
	if n.rawFields == nil {
		type plain KBNote
		raw, _ := json.Marshal(plain(*n))
		_ = json.Unmarshal(raw, &n.rawFields)
	}
	delete(n.rawFields, "content")
	n.Content = ""
	quoteState := "unavailable"
	if _, present := n.rawFields["ref_content"]; present {
		quoteState = "full"
	}
	if n.ContentState != nil {
		quoteState = n.ContentState.RefContent
	}
	n.ContentState = &NoteContentState{Content: "omitted", RefContent: quoteState}
	n.rawFields["content_state"], _ = json.Marshal(n.ContentState)
}
