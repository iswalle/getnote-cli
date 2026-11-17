package client

import (
	"encoding/json"
	"testing"
)

func TestChapterEndOnlyFromNextValidStart(t *testing.T) {
	raw := json.RawMessage(`{"source":"rules","items":[{"start_ms":1000,"title":"A","unknown":"keep"},{"start_ms":2000},{"start_ms":3000,"end_ms":3500},{"start_ms":5000}]}`)
	var got struct {
		Items []map[string]any `json:"items"`
	}
	if json.Unmarshal(ChapterEndTimes(raw), &got) != nil {
		t.Fatal("invalid JSON")
	}
	if got.Items[0]["end_ms"] != float64(2000) || got.Items[0]["end_source"] != "next_chapter_start" || got.Items[0]["unknown"] != "keep" || got.Items[2]["end_ms"] != float64(3500) || got.Items[3]["end_ms"] != nil {
		t.Fatalf("wrong chapter boundaries: %+v", got)
	}
}
