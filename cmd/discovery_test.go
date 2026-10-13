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
		{[]string{"notes", "--source", "dedao", "--note-type", "ref", "--reference-type", "course_article", "--reference-id", "42", "--article-url", "https://example.invalid/a?x=1#p", "--created-from", "2026-10-09T11:00:00+08:00", "--page-size", "7"}, map[string]string{"source": "dedao", "note_type": "ref", "reference_resource_type": "course_article", "reference_resource_id": "42", "reference_article_url": "https://example.invalid/a?x=1#p", "created_from": "2026-10-09T03:00:00Z", "page_size": "7"}},
		{[]string{"kb", "topic-x", "--note-type", "ref", "--page-size", "7", "--page", "2"}, map[string]string{"topic_id": "topic-x", "note_type": "ref", "page": "2", "page_size": "7"}},
		{[]string{"kbs", "--scope", "BOOKSPACE", "--query", "《书名》", "--author", "作者", "--book-id", "book-x", "--page-size", "7"}, map[string]string{"scope": "BOOKSPACE", "query": "书名", "author": "作者", "book_id": "book-x", "page_size": "7"}},
		{[]string{"kbs-sub", "--scope", "BOOKSPACE", "--query", "书名", "--page-size", "7"}, map[string]string{"scope": "BOOKSPACE", "query": "书名", "page_size": "7"}},
		{[]string{"note", "42"}, map[string]string{"id": "42"}},
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
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": []string{"note.reference.v1", "note.filters.v1", "pagination.v1", "knowledge.lookup.v1"}}, "data": map[string]any{"notes": []any{}, "topics": []any{}, "note": map[string]any{"note_id": "42", "content_state": map[string]string{"content": "omitted", "ref_content": "omitted"}}, "pagination": map[string]any{"mode": mode, "page": page, "page_size": 7, "returned_count": 0, "has_more": false, "total_count": nil, "next_page": nil, "next_cursor": nil}, "applied_filters": applied}})
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

func TestDiscoveryLegacyFlagsStillWorkOnOldServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "" {
			t.Error("legacy request gained a new page_size")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"notes":[{"note_id":"42","content":"legacy text"}],"has_more":false,"total":1}}`))
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	for _, args := range [][]string{{"notes", "--limit", "5"}, {"notes", "--cursor", "legacy-opaque"}, {"kb", "topic", "--limit", "5"}, {"notes", "--all"}, {"kb", "topic", "--all"}, {"kb", "topic", "--no-content"}} {
		root := discoveryRoot()
		root.SetArgs(append(args, "-o", "json"))
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		if err := root.Execute(); err != nil {
			t.Errorf("%v: %v", args, err)
		}
		if strings.Contains(strings.Join(args, " "), "--no-content") && strings.Contains(output.String(), `"content": "legacy text"`) {
			t.Error("no-content retained body")
		}
	}
}

func TestDiscoveryFullQuoteTableDoesNotTruncate(t *testing.T) {
	comment, quote := strings.Repeat("评", 210), strings.Repeat("引", 220)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"note": map[string]any{"note_id": "42", "note_type": "ref", "content": comment, "ref_content": quote}}})
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
	if !strings.Contains(output.String(), comment) || !strings.Contains(output.String(), quote) {
		t.Error("full comment or quote truncated")
	}
}

func TestDiscoveryTopicOnlyCannotFallBackToUnfilteredLegacy(t *testing.T) {
	for _, args := range [][]string{{"notes", "--topic-id", "topic-x"}, {"notes", "--topic-id", "topic-x", "--all"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("topic_id") != "topic-x" || r.URL.Query().Get("page_size") != "20" {
					t.Errorf("topic-only request lost its verified scope: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"meta":{"capabilities":["note.filters.v1","pagination.v1"]},"data":{"notes":[],"applied_filters":{"topic_id":"topic-x"},"pagination":{"mode":"cursor","page":null,"page_size":20,"returned_count":0,"has_more":false,"total_count":null,"next_page":null,"next_cursor":null}}}`))
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			root := discoveryRoot()
			root.SetArgs(append(args, "-o", "json"))
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

func TestObsoleteViewFlagDoesNotSendRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("obsolete option reached server")
		w.Write([]byte(`{"success":true,"data":{"notes":[],"note":{"note_id":"42"}}}`))
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	for _, args := range [][]string{{"notes", "--view", "summary"}, {"note", "42", "--view", "full"}, {"kb", "topic", "--view", "summary"}} {
		root := discoveryRoot()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
			t.Errorf("obsolete option was not rejected: %v", err)
		}
	}
}
