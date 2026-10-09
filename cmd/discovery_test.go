package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iswalle/getnote-cli/cmd/kb"
	"github.com/iswalle/getnote-cli/cmd/kbs"
	"github.com/iswalle/getnote-cli/cmd/kbssub"
	"github.com/iswalle/getnote-cli/cmd/note"
	"github.com/iswalle/getnote-cli/cmd/notes"
	"github.com/spf13/cobra"
)

func TestDiscoveryFlagsReachHTTP(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want map[string]string
	}{
		{[]string{"notes", "--view", "summary", "--source", "dedao", "--note-type", "ref", "--reference-type", "course_article", "--reference-id", "42", "--article-url", "https://example.invalid/a?x=1#p", "--created-from", "2026-10-09T11:00:00+08:00", "--page-size", "7"}, map[string]string{"view": "summary", "source": "dedao", "note_type": "ref", "reference_resource_type": "course_article", "reference_resource_id": "42", "reference_article_url": "https://example.invalid/a?x=1#p", "created_from": "2026-10-09T03:00:00Z", "page_size": "7"}},
		{[]string{"kb", "topic-x", "--note-type", "ref", "--view", "summary", "--page-size", "7", "--page", "2"}, map[string]string{"topic_id": "topic-x", "note_type": "ref", "view": "summary", "page": "2", "page_size": "7"}},
		{[]string{"kbs", "--scope", "BOOKSPACE", "--query", "《书名》", "--author", "作者", "--book-id", "book-x", "--page-size", "7"}, map[string]string{"scope": "BOOKSPACE", "query": "书名", "author": "作者", "book_id": "book-x", "page_size": "7"}},
		{[]string{"kbs-sub", "--scope", "BOOKSPACE", "--query", "书名", "--page-size", "7"}, map[string]string{"scope": "BOOKSPACE", "query": "书名", "page_size": "7"}},
		{[]string{"note", "42", "--view", "summary"}, map[string]string{"id": "42", "view": "summary"}},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				q := r.URL.Query()
				for k, v := range tc.want {
					if q.Get(k) != v {
						t.Errorf("%s=%q want %q", k, q.Get(k), v)
					}
				}
				applied := map[string]string{}
				for k, values := range q {
					if k != "view" && k != "page" && k != "page_size" && k != "id" {
						applied[k] = values[0]
					}
				}
				page := 1
				if q.Get("page") == "2" {
					page = 2
				}
				mode := "page"
				if tc.args[0] == "notes" {
					mode = "cursor"
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": []string{"note.reference.v1", "note.summary.v1", "note.filters.v1", "pagination.v1", "knowledge.lookup.v1"}}, "data": map[string]any{"notes": []any{}, "topics": []any{}, "note": map[string]any{"note_id": "42", "content_state": map[string]string{"content": "omitted", "ref_content": "omitted"}}, "pagination": map[string]any{"mode": mode, "page": page, "page_size": 7, "returned_count": 0, "has_more": false, "total_count": nil, "next_page": nil, "next_cursor": nil}, "applied_filters": applied}})
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			root := discoveryRoot()
			root.SetArgs(append(tc.args, "-o", "json"))
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Errorf("requests=%d", calls)
			}
		})
	}
}

func TestDiscoveryReferenceTableSeparatesCommentAndQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"note":{"note_id":"42","title":"","display_title":"示例课节","note_type":"ref","content":"个人评论","ref_content":"完整引用原文"}}}`))
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	root := discoveryRoot()
	root.SetArgs([]string{"note", "42"})
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"示例课节", "个人评论", "完整引用原文", "Comment", "Quote"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("table missing %q: %s", want, output.String())
		}
	}
}

func discoveryRoot() *cobra.Command {
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.AddCommand(notes.NewNotesCmd(), note.NewNoteCmd(), kb.NewKbCmd(), kbs.NewKbsCmd(), kbssub.NewKbsSubCmd())
	return root
}
