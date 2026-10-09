package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoveryTraversalCompletesOrClearlyFails(t *testing.T) {
	for _, kind := range []string{"notes", "kb", "lookup"} {
		for _, mode := range []string{"41 rows", "duplicate", "repeated continuation", "empty intermediate", "later failure", "empty final", "duplicate final"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if calls > 3 {
						t.Error("unbounded traversal")
						http.Error(w, "stop", 500)
						return
					}
					if calls == 2 && mode == "later failure" {
						http.Error(w, "transient failure", 503)
						return
					}
					q := r.URL.Query()
					if kind == "notes" {
						expected := ""
						if calls > 1 {
							expected = fmt.Sprint("cursor-", calls)
						}
						if q.Get("cursor") != expected {
							t.Errorf("cursor=%q want %q", q.Get("cursor"), expected)
						}
					} else if q.Get("page") != fmt.Sprint(calls) {
						t.Errorf("page=%q", q.Get("page"))
					}
					count := 20
					if calls == 3 {
						count = 1
					}
					if calls == 2 && (mode == "empty intermediate" || mode == "empty final") {
						count = 0
					}
					items := []any{}
					for i := 0; i < count; i++ {
						id := fmt.Sprint(100 + (calls-1)*20 + i)
						if calls == 2 && mode == "duplicate final" {
							id = fmt.Sprint(100 + i)
						}
						if calls == 2 && i == 0 && mode == "duplicate" {
							id = "119"
						}
						items = append(items, map[string]any{"note_id": id, "topic_id": id, "title": "test"})
					}
					pagination := map[string]any{"mode": "cursor", "page_size": 20, "returned_count": count, "total_count": nil, "has_more": calls < 3, "next_cursor": nil, "next_page": nil}
					if calls == 2 && (mode == "empty final" || mode == "duplicate final") {
						pagination["has_more"] = false
					}
					if kind == "notes" {
						if pagination["has_more"] == true {
							pagination["next_cursor"] = fmt.Sprint("cursor-", calls+1)
							if calls == 2 && mode == "repeated continuation" {
								pagination["next_cursor"] = "cursor-2"
							}
						}
					} else {
						pagination["mode"] = "page"
						pagination["page"] = calls
						if pagination["has_more"] == true {
							pagination["next_page"] = calls + 1
							if calls == 2 && mode == "repeated continuation" {
								pagination["next_page"] = 2
							}
						}
					}
					filters := map[string]string{}
					if kind == "kb" {
						filters["topic_id"] = "topic-x"
					}
					if kind == "lookup" {
						filters["scope"] = "BOOKSPACE"
					}
					data := map[string]any{"notes": items, "topics": items, "pagination": pagination, "applied_filters": filters}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": []string{"pagination.v1", "note.filters.v1", "knowledge.lookup.v1"}}, "data": data})
				}))
				defer server.Close()
				t.Setenv("GETNOTE_API_URL", server.URL)
				var count int
				var traversal *Traversal
				var err error
				c := New("")
				switch kind {
				case "notes":
					var r *NoteListResponse
					r, err = c.AllNotes(DiscoveryOptions{})
					if r != nil {
						count = len(r.Data.Notes)
						traversal = r.Data.Traversal
					}
				case "kb":
					var r *KBNotesResponse
					r, err = c.AllKBNotes("topic-x", DiscoveryOptions{})
					if r != nil {
						count = len(r.Data.Notes)
						traversal = r.Data.Traversal
					}
				case "lookup":
					var r *KBListResponse
					r, err = c.AllKnowledge(KnowledgeLookupOptions{Scope: "BOOKSPACE"}, false)
					if r != nil {
						count = len(r.Data.Topics)
						traversal = r.Data.Traversal
					}
				}
				if mode == "41 rows" || mode == "duplicate" {
					if err != nil {
						t.Fatal(err)
					}
					want := 41
					if mode == "duplicate" {
						want = 40
					}
					if count != want || calls != 3 || traversal == nil || !traversal.Completed || traversal.ReturnedCount != want {
						t.Errorf("count=%d calls=%d traversal=%+v", count, calls, traversal)
					}
				} else if err == nil || !strings.Contains(err.Error(), "未完成") {
					t.Errorf("must report incomplete traversal, got %v", err)
				}
			})
		}
	}
}
