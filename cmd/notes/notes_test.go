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
