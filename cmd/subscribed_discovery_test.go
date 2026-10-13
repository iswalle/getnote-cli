package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// Subscription lookup adds the boolean exclude_mine alongside string selectors.
func TestSubscribedDiscoveryTypedFilters(t *testing.T) {
	for _, mode := range []string{"single", "all", "empty", "missing author", "wrong author", "boolean author", "null author", "later filter lost"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				q := r.URL.Query()
				if r.URL.Path != "/open/api/v1/resource/knowledge/subscribe/list" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				for key, want := range map[string]string{"scope": "BOOKSPACE", "author": "博尔赫斯", "query": "沙之书", "page_size": "1", "page": strconv.Itoa(calls)} {
					if q.Get(key) != want {
						t.Errorf("%s=%q want %q", key, q.Get(key), want)
					}
				}
				filters := map[string]any{"scope": "BOOKSPACE", "author": "博尔赫斯", "query": "沙之书", "exclude_mine": true}
				switch mode {
				case "missing author":
					delete(filters, "author")
				case "wrong author":
					filters["author"] = "其他作者"
				case "boolean author":
					filters["author"] = true
				case "null author":
					filters["author"] = nil
				case "later filter lost":
					if calls == 2 {
						delete(filters, "author")
					}
				}
				topics := []any{map[string]any{"topic_id": strconv.Itoa(calls), "name": "沙之书"}}
				if mode == "empty" {
					topics = []any{}
				}
				more := (mode == "all" || mode == "later filter lost") && calls == 1
				var next any
				if more {
					next = 2
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": []string{"pagination.v1", "knowledge.lookup.v1"}}, "data": map[string]any{
					"topics": topics, "applied_filters": filters,
					"pagination": map[string]any{"mode": "page", "page": calls, "page_size": 1, "returned_count": len(topics), "has_more": more, "next_page": next},
				}})
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			root := discoveryRoot()
			args := []string{"kbs-sub", "--scope", "BOOKSPACE", "--author", "博尔赫斯", "--query", "沙之书", "--page-size", "1", "-o", "json"}
			if mode != "single" {
				args = append(args, "--all")
			}
			root.SetArgs(args)
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			err := root.Execute()
			wantError := mode != "single" && mode != "all" && mode != "empty"
			if wantError {
				if err == nil || !strings.Contains(err.Error(), "server did not apply author") {
					t.Fatalf("expected rejected author filter, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Data struct {
					AppliedFilters map[string]json.RawMessage `json:"applied_filters"`
					Topics         []json.RawMessage          `json:"topics"`
					Traversal      struct {
						Completed bool `json:"completed"`
						Pages     int  `json:"pages"`
					} `json:"traversal"`
				} `json:"data"`
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if string(result.Data.AppliedFilters["exclude_mine"]) != "true" {
				t.Fatalf("lost boolean filter: %s", output.Bytes())
			}
			wantCount, wantCalls := 1, 1
			if mode == "all" {
				wantCount, wantCalls = 2, 2
			}
			if mode == "empty" {
				wantCount = 0
			}
			if len(result.Data.Topics) != wantCount || calls != wantCalls {
				t.Fatalf("got %d topics, %d calls", len(result.Data.Topics), calls)
			}
			if mode != "single" && (!result.Data.Traversal.Completed || result.Data.Traversal.Pages != wantCalls) {
				t.Fatalf("incorrect traversal: %s", output.Bytes())
			}
		})
	}
}
