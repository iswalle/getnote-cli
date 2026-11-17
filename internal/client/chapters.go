package client

import (
	"encoding/json"
)

// ChapterEndTimes adds only the immediate next chapter's start in milliseconds.
// Existing values and unknown fields survive; the final chapter remains unknown.
func ChapterEndTimes(raw json.RawMessage) json.RawMessage {
	var timeline map[string]json.RawMessage
	var items []map[string]json.RawMessage
	if json.Unmarshal(raw, &timeline) != nil || json.Unmarshal(timeline["items"], &items) != nil {
		return raw
	}
	for i, item := range items {
		if item == nil {
			continue
		}
		if _, ok := item["end_ms"]; ok {
			continue
		}
		item["end_ms"] = json.RawMessage("null")
		var start, end int64
		if json.Unmarshal(item["start_ms"], &start) != nil || string(item["start_ms"]) == "null" || start < 0 || i+1 >= len(items) {
			continue
		}
		if json.Unmarshal(items[i+1]["start_ms"], &end) == nil && end > start {
			item["end_ms"] = items[i+1]["start_ms"]
			item["end_source"] = json.RawMessage(`"next_chapter_start"`)
		}
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return raw
	}
	timeline["items"] = encoded
	out, err := json.Marshal(timeline)
	if err != nil {
		return raw
	}
	return out
}
