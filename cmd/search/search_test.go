package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestIsSearchTimeout(t *testing.T) {
	t.Parallel()
	if !isSearchTimeout(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded must be recognized as timeout")
	}
	if isSearchTimeout(errors.New("search failed")) {
		t.Fatal("ordinary error must not be recognized as timeout")
	}
}

func TestPartialEmptySearchIsNotReportedAsNoMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":{"results":[],"partial":true,"warnings":[{"code":"file_identity_unavailable","resource_type":"file","count":1}]}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	var out bytes.Buffer
	cmd := NewSearchCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"query"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Partial search") || !strings.Contains(out.String(), "file_identity_unavailable") || strings.Contains(out.String(), "No results found.") {
		t.Fatalf("partial search was hidden: %s", out.String())
	}
}

func TestExperienceSnippetCommandDisplaysActualSnippet(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["search.snippet.v1"]},"data":{"response_mode":"snippet","max_snippet_chars":400,"results":[{"resource_type":"file","resource_id":"7","title":"file","snippet":"actual snippet","content_state":{"content":"omitted","snippet":"full"},"score":null}]}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	cmd := NewSearchCmd()
	cmd.SetArgs([]string{"query", "--response-mode", "snippet"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "actual snippet") {
		t.Fatalf("snippet hidden: %s", out.String())
	}
}

func TestTerminalSearchPreservesFullNoteURL(t *testing.T) {
	const noteID = "1234567890123456789"
	const productionURL = "https://www.biji.com/note/" + noteID
	const serverURL = "https://notes.example.com/workspaces/example/note/" + noteID + "?source=cli&view=original#details"
	title := strings.Repeat("示", 25)

	for _, tc := range []struct {
		name    string
		noteURL string
		webURL  string
		wantURL string
	}{
		{name: "production URL", wantURL: productionURL},
		{name: "server URL with query and fragment", noteURL: serverURL, wantURL: serverURL},
		{name: "custom web URL fallback", webURL: "https://getnote.staging.example.com/app/", wantURL: "https://getnote.staging.example.com/app/note/" + noteID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"success":true,"data":{"results":[{"resource_type":"note","note_id":%q,"title":%q,"note_type":"plain_text","created_at":"2026-01-01 12:00:00","content":"Example content","score":0.9,"note_url":%q}]}}`, noteID, title, tc.noteURL)
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			t.Setenv("GETNOTE_API_KEY", "test-api-key")
			t.Setenv("GETNOTE_CLIENT_ID", "test-client-id")
			t.Setenv("GETNOTE_WEB_URL", tc.webURL)

			root := &cobra.Command{Use: "getnote"}
			root.PersistentFlags().StringP("output", "o", "table", "")
			root.AddCommand(NewSearchCmd())
			root.SetArgs([]string{"search", "example"})
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}

			var row string
			for _, line := range strings.Split(out.String(), "\n") {
				if strings.HasPrefix(line, noteID) {
					row = line
					break
				}
			}
			if !strings.HasSuffix(strings.TrimSpace(row), tc.wantURL) {
				t.Errorf("search row does not contain the complete URL %q: %s", tc.wantURL, row)
			}
			if strings.Contains(row, title) || !strings.Contains(row, strings.Repeat("示", 20)+"…") {
				t.Errorf("title should retain its existing truncation: %s", row)
			}
		})
	}
}
