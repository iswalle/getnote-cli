package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFileAndReportRoutesPreserveMetadataAndPagination(t *testing.T) {
	var paths []string
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{"status":"UPLOADING"},"request_id":"test"}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	c := New("")
	for _, call := range []func() (*ResourceResponse, error){
		func() (*ResourceResponse, error) { return c.NoteMarks("1922071641760757698") },
		func() (*ResourceResponse, error) { return c.SproutsPage("2026-09", "1922071641760757698", 2) },
		func() (*ResourceResponse, error) { return c.Sprout("report-alias") },
		c.FileCapabilities,
		func() (*ResourceResponse, error) { return c.FileToken("HTM") },
		func() (*ResourceResponse, error) {
			return c.FileAdd(map[string]string{"directory_id": "9000000000020341", "file_type": "HTM"})
		},
	} {
		r, err := call()
		if err != nil || !r.Success || r.RequestID != "test" {
			t.Fatalf("unexpected response: %#v %v", r, err)
		}
	}
	want := []string{
		"/open/api/v1/resource/note/marks?note_id=1922071641760757698",
		"/open/api/v1/resource/note/sprouts?limit=2&month=2026-09&since_id=1922071641760757698",
		"/open/api/v1/resource/note/sprout?id=report-alias",
		"/open/api/v1/resource/knowledge/file/capabilities",
		"/open/api/v1/resource/knowledge/file/upload_token?mime_type=HTM",
		"/open/api/v1/resource/knowledge/file/upload",
	}
	if strings.Join(paths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected routes: %v", paths)
	}
	if payload["directory_id"] != "9000000000020341" || payload["file_type"] != "HTM" {
		t.Fatal("metadata changed")
	}
}

func TestTimelineAndResourceFieldsSurviveClientDecoding(t *testing.T) {
	var note Note
	err := json.Unmarshal([]byte(`{"audio":{"sentences":[{"speaker_id":3,"speaker_name":"测试说话人","text":"hello"}]},"timeline":{"schema_version":2,"moments":[{"id":"photo1","type":"photo","action_time":14000,"content":"caption","files":[{"url":"https://example.com/photo.jpg"}]}]}}`), &note)
	if err != nil {
		t.Fatal(err)
	}
	if note.Timeline.SchemaVersion != 2 || note.Timeline.Moments[0].Type != "photo" || note.Timeline.Moments[0].ActionTime != 14000 {
		t.Fatal("timeline metadata lost")
	}
	encoded, _ := json.Marshal(note)
	for _, field := range []string{"speaker_name", "测试说话人", "photo.jpg", "caption"} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("lost %s", field)
		}
	}
	var file KBResource
	if err := json.Unmarshal([]byte(`{"original_url":"https://example.com/a.doc","preview_url":"https://example.com/a.pdf","fail_reason":"failed"}`), &file); err != nil {
		t.Fatal(err)
	}
	if file.OriginalURL == "" || file.PreviewURL == "" || file.FailReason != "failed" {
		t.Fatal("resource fields lost")
	}
}
