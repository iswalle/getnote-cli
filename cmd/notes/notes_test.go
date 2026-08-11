package notes

import (
	"testing"

	"github.com/iswalle/getnote-cli/internal/client"
)

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
