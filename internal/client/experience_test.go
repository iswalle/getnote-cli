package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExperienceCourseRequiresPairAndCapability(t *testing.T) {
	for _, mode := range []string{"supported", "old", "ignored"} {
		t.Run(mode, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if q.Get("reference_course_type") != "66" || q.Get("reference_course_id") != "9007199254740993" {
					t.Error("lost course identity")
				}
				caps := []string{"note.filters.v1", "pagination.v1", "future.v2"}
				if mode != "old" {
					caps = append(caps, "note.course_filter.v1")
				}
				filters := map[string]string{"source": "dedao", "reference_course_type": "66", "reference_course_id": "9007199254740993"}
				if mode == "ignored" {
					delete(filters, "reference_course_id")
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": caps}, "data": map[string]any{"notes": []any{}, "applied_filters": filters, "pagination": map[string]any{"mode": "cursor", "page_size": 20, "returned_count": 0, "has_more": false}}})
			}))
			defer s.Close()
			t.Setenv("GETNOTE_API_URL", s.URL)
			_, err := New("").NoteListWithOptions(DiscoveryOptions{Source: "dedao", CourseType: "66", CourseID: "9007199254740993"})
			if (err == nil) != (mode == "supported") {
				t.Fatalf("mode=%s err=%v", mode, err)
			}
		})
	}
	for _, o := range []DiscoveryOptions{{CourseType: "66"}, {CourseID: "1"}, {Source: "dedao", CourseType: "66", CourseID: "1", TopicID: "kb"}, {Source: "dedao", CourseType: "66", CourseID: "9223372036854775808"}} {
		if _, _, err := o.values(); err == nil {
			t.Fatalf("invalid course options accepted: %+v", o)
		}
	}
}

func TestExperienceReferenceWalkContinuesEmptyPagesAndRejectsCycles(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		t.Run(fmt.Sprint(cycle), func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				q := r.URL.Query()
				if r.URL.Path != "/open/api/v1/resource/reference/list" || q.Get("query") != "课程" || q.Get("match_mode") != "exact" {
					t.Error("wrong reference request")
				}
				more := calls == 1 || cycle
				var next any
				if more {
					next = "next"
				}
				rows := []any{}
				if calls > 1 && !cycle {
					rows = append(rows, map[string]any{"course_type": "66", "course_id": "9007199254740993", "source": "dedao", "resource_type": "course", "has_notes": true, "metadata_state": "future_state", "future_field": true})
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": []string{"reference.resources.v1", "pagination.v1", "future.v2"}}, "data": map[string]any{"resources": rows, "applied_filters": map[string]string{"source": "dedao", "resource_type": "course", "query": "课程", "match_mode": "exact"}, "pagination": map[string]any{"mode": "cursor", "page_size": 20, "returned_count": len(rows), "has_more": more, "next_cursor": next}}})
			}))
			defer s.Close()
			t.Setenv("GETNOTE_API_URL", s.URL)
			r, err := New("").AllReferences(ReferenceLookupOptions{Source: "dedao", ResourceType: "course", Query: "课程"})
			if cycle {
				if err == nil || calls > 2 {
					t.Fatalf("cycle accepted calls=%d err=%v", calls, err)
				}
				return
			}
			if err != nil || calls != 2 || len(r.Data.Resources) != 1 || r.Data.Resources[0].CourseID != "9007199254740993" || r.Data.Traversal == nil || !r.Data.Traversal.Completed {
				t.Fatalf("empty page ended walk: calls=%d err=%v", calls, err)
			}
			raw, _ := json.Marshal(r)
			if !strings.Contains(string(raw), `"future_field":true`) {
				t.Fatal("unknown field lost")
			}
		})
	}
}

func TestExperienceBatchIsOneRequestAndPreservesStatuses(t *testing.T) {
	for _, supported := range []bool{false, true} {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != "POST" || r.URL.Path != "/open/api/v1/resource/note/batch-detail" {
				t.Error("batch fanned out")
			}
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)
			if fmt.Sprint(req["ids"]) != "[9007199254740993 2]" {
				t.Errorf("IDs changed: %v", req["ids"])
			}
			caps := []string{}
			if supported {
				caps = append(caps, "note.batch_detail.v1")
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": caps}, "data": map[string]any{"items": []any{map[string]any{"note_id": "9007199254740993", "status": "ok", "note": map[string]any{"note_id": "9007199254740993", "original": map[string]any{"source": "ref_content", "content": "中😀"}, "future_field": true}}, map[string]any{"note_id": "2", "status": "deferred_budget"}}, "pending_ids": []string{"2"}, "partial": true, "max_total_chars": 32000, "max_total_bytes": 1048576, "total_chars": 2}})
		}))
		t.Setenv("GETNOTE_API_URL", s.URL)
		r, err := New("").NotesBatch(BatchDetailOptions{IDs: []string{"9007199254740993", "0002", "2"}, Fields: []string{"original"}})
		s.Close()
		if calls != 1 || (err == nil) != supported {
			t.Fatalf("calls=%d supported=%t err=%v", calls, supported, err)
		}
		if supported {
			raw, _ := json.Marshal(r)
			if !strings.Contains(string(raw), `"future_field":true`) || len(r.Data.PendingIDs) != 1 || r.Data.Items[1].Status != "deferred_budget" {
				t.Fatal("batch response lost state")
			}
		}
	}
}

func TestExperienceSnippetOptInKeepsOmittedBodyAndUnknownFields(t *testing.T) {
	for _, supported := range []bool{false, true} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)
			if req["response_mode"] != "snippet" || req["max_snippet_chars"] != float64(100) {
				t.Error("snippet request lost")
			}
			caps := []string{}
			if supported {
				caps = append(caps, "search.snippet.v1")
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": caps}, "data": map[string]any{"response_mode": "snippet", "max_snippet_chars": 100, "max_total_bytes": 1048576, "results": []any{map[string]any{"resource_type": "future_kind", "snippet": "中😀", "content_state": map[string]string{"content": "omitted", "snippet": "full"}, "truncated": false, "future_field": true}}}})
		}))
		t.Setenv("GETNOTE_API_URL", s.URL)
		n := 100
		r, err := New("").SearchWithOptions("", "AI", 3, SearchOptions{ResponseMode: "snippet", MaxSnippetChars: &n})
		s.Close()
		if (err == nil) != supported {
			t.Fatalf("err=%v supported=%t", err, supported)
		}
		if supported {
			raw, _ := json.Marshal(r)
			var wire map[string]any
			json.Unmarshal(raw, &wire)
			item := wire["data"].(map[string]any)["results"].([]any)[0].(map[string]any)
			if _, exists := item["content"]; exists || item["future_field"] != true || r.Data.Results[0].Snippet != "中😀" {
				t.Fatal("snippet reconstructed old content or lost fields")
			}
		}
	}
}

func TestExperienceReferenceWalkKeepsFirstPage(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["reference.resources.v1","pagination.v1"]},"data":{"resources":[{"course_type":"66","course_id":"84"}],"applied_filters":{"source":"dedao","resource_type":"course","query":"course","match_mode":"exact"},"pagination":{"mode":"cursor","page_size":20,"returned_count":1,"has_more":false}}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	r, err := New("").AllReferences(ReferenceLookupOptions{Source: "dedao", ResourceType: "course", Query: "course"})
	if err != nil || len(r.Data.Resources) != 1 {
		t.Fatalf("first page discarded: %v %+v", err, r)
	}
}

func TestExperienceNotePreservesCourseAndUnknownReferenceMetadata(t *testing.T) {
	var note Note
	json.Unmarshal([]byte(`{"note_id":"9007199254740993","reference":{"resource_id":"22961","course":{"type":"66","id":"84","title":"course","future_course":true},"future_reference":true}}`), &note)
	raw, err := json.Marshal(note)
	if err != nil || !strings.Contains(string(raw), `"future_reference":true`) || !strings.Contains(string(raw), `"future_course":true`) || !strings.Contains(string(raw), `"course"`) {
		t.Fatalf("course metadata lost: %s err=%v", raw, err)
	}
}
