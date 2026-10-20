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

func TestSproutsRejectsInvalidMonthBeforeSendingRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"success":true,"data":{"tasks":[]}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	c := New("")
	for _, month := range []string{"2026-00", "2026-13", "2026-99", "2026-1", "26-01", "2026-01-01", " 2026-01", "garbage"} {
		if _, err := c.SproutsPage(month, "", 20); err == nil {
			t.Errorf("accepted invalid month %q", month)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid months made %d HTTP requests", calls)
	}
	for _, month := range []string{"2026-01", "2024-02", "2026-12"} {
		if _, err := c.Sprouts(month); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatalf("valid months made %d requests", calls)
	}
}

func TestFileAddKeepsNumericIDsAndPublishesLosslessStringAliases(t *testing.T) {
	for _, raw := range []string{
		`{"id":9007199254740993,"topic_id":9223372036854775807,"directory_id":9007199254740995,"status":"UPLOADING","future":{"v":9007199254740997}}`,
		`{"id":9007199254740993,"id_str":"9007199254740993","topic_id":9223372036854775807,"topic_id_str":"9223372036854775807","directory_id":9007199254740995,"directory_id_str":"9007199254740995","status":"UPLOADING","future":{"v":9007199254740997}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"success":true,"data":`+raw+`}`)
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			got, err := New("").FileAdd(map[string]string{"topic_id": "test"})
			if err != nil {
				t.Fatal(err)
			}
			var out map[string]json.RawMessage
			if err := json.Unmarshal(got.Data, &out); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]string{"id": "9007199254740993", "topic_id": "9223372036854775807", "directory_id": "9007199254740995", "id_str": `"9007199254740993"`, "topic_id_str": `"9223372036854775807"`, "directory_id_str": `"9007199254740995"`, "future": `{"v":9007199254740997}`} {
				if string(out[key]) != want {
					t.Errorf("%s=%s want %s", key, out[key], want)
				}
			}
		})
	}
}

func TestFileAddRejectsInconsistentIDStrings(t *testing.T) {
	for _, raw := range []string{`{"id":9007199254740993,"id_str":"9007199254740992"}`, `{"id":null}`, `{"id":1.25}`, `{"id":9223372036854775808}`} {
		t.Run(raw, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"success":true,"data":`+raw+`}`)
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			if _, err := New("").FileAdd(map[string]string{}); err == nil {
				t.Fatal("invalid file identity accepted")
			}
		})
	}
}
