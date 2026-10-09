package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoveryRequestsAndCapabilityProof(t *testing.T) {
	for _, kind := range []string{"notes", "kb", "lookup", "subscribed", "detail"} {
		for _, mode := range []string{"supported", "old", "ignored"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					applied := map[string]string{}
					for k, v := range q {
						if k != "page_size" && k != "page" && k != "cursor" && k != "view" && k != "id" {
							applied[k] = v[0]
						}
					}
					if kind == "lookup" || kind == "subscribed" {
						for k, v := range map[string]string{"query": "书名", "scope": "BOOKSPACE", "author": "作者", "book_id": "book-x", "page_size": "7", "page": "2"} {
							if q.Get(k) != v {
								t.Errorf("query %s=%q want %q", k, q.Get(k), v)
							}
						}
					} else if kind == "detail" {
						if q.Get("view") != "summary" || q.Get("id") != "9007199254740993" {
							t.Errorf("detail query: %v", q)
						}
					} else {
						for k, v := range map[string]string{"source": "dedao", "note_type": "ref", "reference_resource_type": "course_article", "reference_resource_id": "9007199254740993", "reference_article_url": "https://example.invalid/a?x=1#p", "created_from": "2026-10-09T03:00:00Z", "edited_to": "2026-10-10T03:00:00Z", "record_updated_from": "2026-10-08T03:00:00Z", "page_size": "7", "view": "summary"} {
							if q.Get(k) != v {
								t.Errorf("query %s=%q want %q", k, q.Get(k), v)
							}
						}
						if kind == "kb" && q.Get("topic_id") != "topic-x" {
							t.Errorf("topic missing: %v", q)
						}
					}
					if mode == "ignored" {
						applied = map[string]string{}
					}
					caps := []string{"note.filters.v1", "note.summary.v1", "note.reference.v1", "pagination.v1", "knowledge.lookup.v1", "note.timestamps.v1"}
					if mode == "old" {
						caps = nil
					}
					data := map[string]any{"notes": []any{}, "topics": []any{}, "note": map[string]any{"note_id": "9007199254740993", "content_state": map[string]string{"content": "omitted", "ref_content": "omitted"}}, "pagination": map[string]any{"mode": "cursor", "page_size": 7, "returned_count": 0, "has_more": false, "total_count": nil, "next_cursor": nil, "next_page": nil}, "applied_filters": applied}
					if kind == "kb" || kind == "lookup" || kind == "subscribed" {
						data["pagination"].(map[string]any)["mode"] = "page"
						data["pagination"].(map[string]any)["page"] = 2
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": caps}, "data": data})
				}))
				defer server.Close()
				t.Setenv("GETNOTE_API_URL", server.URL)
				c := New("")
				var err error
				o := DiscoveryOptions{Source: "dedao", NoteType: "ref", ReferenceType: "course_article", ReferenceID: "9007199254740993", ArticleURL: "https://example.invalid/a?x=1#p", CreatedFrom: "2026-10-09T11:00:00+08:00", EditedTo: "2026-10-10T11:00:00+08:00", RecordUpdatedFrom: "2026-10-08T11:00:00+08:00", PageSize: 7, View: "summary"}
				switch kind {
				case "notes":
					_, err = c.NoteListWithOptions(o)
				case "kb":
					o.Page = 2
					_, err = c.KBNotesWithOptions("topic-x", o)
				case "lookup", "subscribed":
					_, err = c.KBListWithOptions(KnowledgeLookupOptions{Scope: "BOOKSPACE", Query: "  《书名》  ", Author: "作者", BookID: "book-x", Page: 2, PageSize: 7}, kind == "subscribed")
				case "detail":
					_, err = c.NoteGetWithOptions("9007199254740993", "summary")
				}
				wantError := mode == "old" || (mode == "ignored" && kind != "detail")
				if (err != nil) != wantError {
					t.Errorf("error=%v wantError=%v", err, wantError)
				}
			})
		}
	}
}

func TestDiscoveryInvalidOptionsDoNotSendRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid options sent request")
		fmt.Fprint(w, `{"success":true}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	for _, o := range []DiscoveryOptions{{PageSize: 101}, {PageSize: -1}, {View: "bad"}, {ReferenceID: "42"}, {CreatedFrom: "yesterday"}, {TopicID: "x", Page: 1, Cursor: "c"}} {
		if _, err := New("").NoteListWithOptions(o); err == nil {
			t.Errorf("accepted %+v", o)
		}
	}
}

func TestDiscoveryReferenceSelectorsNeedReferenceCapability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"meta":{"capabilities":["note.filters.v1","pagination.v1"]},"data":{"notes":[],"applied_filters":{"reference_resource_type":"ebook"},"pagination":{"mode":"cursor","page_size":20,"returned_count":0,"total_count":null,"has_more":false,"next_cursor":null,"next_page":null}}}`))
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	if _, err := New("").NoteListWithOptions(DiscoveryOptions{ReferenceType: "ebook"}); err == nil {
		t.Error("unverified reference capability accepted")
	}
}

func TestDiscoveryUnknownLegacyTotalIsNotInvented(t *testing.T) {
	var value NoteListData
	if err := json.Unmarshal([]byte(`{"notes":[],"pagination":{"total_count":null}}`), &value); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	_ = json.Unmarshal(data, &decoded)
	if total, present := decoded["total"]; present && total != nil {
		t.Errorf("invented legacy total: %s", data)
	}
}

func TestDiscoveryNoteRoundTripPreservesMetadataAndOmission(t *testing.T) {
	input := `{"id":"9007199254740993","note_id":"9007199254740993","title":"","display_title":"引用笔记｜测试","note_type":"ref","source":"dedao","reference":{"ptype":0,"resource_id":"9007199254740999","location":{"page":0}},"topics":[{"id":"general","name":"积累"}],"timestamps":{"created_at":"2026-10-09T11:00:00+08:00"},"preview":{"text":"测试","source":"ref_content","truncated":false},"content_state":{"content":"omitted","ref_content":"omitted"},"metadata_state":{"reference":"complete","topics":"complete"},"future_field":{"retained":true}}`
	for _, target := range []any{&Note{}, &KBNote{}} {
		if err := json.Unmarshal([]byte(input), target); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(target)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]json.RawMessage
		if err = json.Unmarshal(encoded, &got); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"display_title", "reference", "topics", "timestamps", "preview", "content_state", "metadata_state", "future_field"} {
			if _, ok := got[key]; !ok {
				t.Errorf("%T lost %s: %s", target, key, encoded)
			}
		}
		if _, ok := got["content"]; ok {
			t.Errorf("%T fabricated omitted content", target)
		}
		if _, ok := got["ref_content"]; ok {
			t.Errorf("%T fabricated omitted quote", target)
		}
		var ref map[string]any
		if err = json.Unmarshal(got["reference"], &ref); err != nil {
			t.Fatal(err)
		}
		if ref["ptype"] != float64(0) || ref["resource_id"] != "9007199254740999" || ref["location"].(map[string]any)["page"] != float64(0) {
			t.Errorf("reference values changed: %s", got["reference"])
		}
	}
}

func TestDiscoverySearchUnknownScoreAndFileIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"meta":{"capabilities":["note.reference.v1"]},"data":{"result_mode":"top_k","is_exhaustive":false,"returned_count":1,"distinct_note_count":0,"results":[{"resource_type":"file","note_type":"FILE","content":"片段","score":null,"snippet_source":"unknown"}]}}`))
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	resp, err := New("").NoteSearch("测试", 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	d := got["data"].(map[string]any)
	hit := d["results"].([]any)[0].(map[string]any)
	if hit["score"] != nil {
		t.Errorf("unknown score became %v", hit["score"])
	}
	if hit["resource_type"] != "file" || d["result_mode"] != "top_k" {
		t.Errorf("search contract lost: %s", b)
	}
	if u, ok := hit["note_url"]; ok && u != "" {
		t.Errorf("file became a note: %s", b)
	}
}
