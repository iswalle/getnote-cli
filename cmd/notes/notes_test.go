package notes

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/spf13/cobra"
)

func TestTerminalNotesShowsIncompleteScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["pagination.v1","note.filters.v1","note.course_filter.v1"]},"data":{"notes":[],"pagination":{"mode":"cursor","page_size":20,"returned_count":0,"total_count":null,"has_more":false,"next_cursor":null},"applied_filters":{"source":"dedao","reference_course_type":"66","reference_course_id":"84"},"partial":true,"warnings":[{"code":"course_directory_no_lessons"}],"coverage":{"kind":"course_reference_only","includes_personal_writing":false}}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.AddCommand(NewNotesCmd())
	root.SetArgs([]string{"notes", "--source", "dedao", "--course-type", "66", "--course-id", "84", "--all"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Partial notes", "course_directory_no_lessons", "course_reference_only", "Traversal incomplete"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("terminal hid %q: %s", want, out.String())
		}
	}
}

func TestApplyLimitKeepsPaginationAtLastVisibleNote(t *testing.T) {
	resp := &client.NoteListResponse{
		Success: true,
		Data: client.NoteListData{
			Notes: []client.Note{
				{ID: client.StringID("1916020531058082912"), NoteID: "1916020531058082912"},
				{ID: client.StringID("1916020531058082913"), NoteID: "1916020531058082913"},
				{ID: client.StringID("1916020531058082914"), NoteID: "1916020531058082914"},
			},
		},
	}

	applyLimit(resp, 2)

	if len(resp.Data.Notes) != 2 || !resp.Data.HasMore {
		t.Fatalf("limited response = %+v", resp.Data)
	}
	if resp.Data.Cursor != "1916020531058082913" ||
		resp.Data.NextCursor.String() != "1916020531058082913" {
		t.Fatalf("cursor = %q / %q", resp.Data.Cursor, resp.Data.NextCursor)
	}
}

func TestTerminalNotesPreservesFullNoteURL(t *testing.T) {
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
				fmt.Fprintf(w, `{"success":true,"data":{"notes":[{"note_id":%q,"title":%q,"note_type":"plain_text","created_at":"2026-01-01 12:00:00","note_url":%q}],"has_more":false}}`, noteID, title, tc.noteURL)
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			t.Setenv("GETNOTE_API_KEY", "test-api-key")
			t.Setenv("GETNOTE_CLIENT_ID", "test-client-id")
			t.Setenv("GETNOTE_WEB_URL", tc.webURL)

			root := &cobra.Command{Use: "getnote"}
			root.PersistentFlags().StringP("output", "o", "table", "")
			root.AddCommand(NewNotesCmd())
			root.SetArgs([]string{"notes", "--limit", "20"})
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
				t.Errorf("note row does not contain the complete URL %q: %s", tc.wantURL, row)
			}
			if strings.Contains(row, title) || !strings.Contains(row, strings.Repeat("示", 19)+"…") {
				t.Errorf("title should retain its existing truncation: %s", row)
			}
		})
	}
}
