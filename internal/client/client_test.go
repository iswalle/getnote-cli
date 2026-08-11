package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP200BusinessFailureReturnsStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": false,
			"data": null,
			"error": {
				"code": 10000,
				"message": "参数错误",
				"reason": "invalid_request",
				"retryable": false,
				"field": "parent_id",
				"constraint": "non_negative_decimal_integer",
				"expected_type": "decimal string or JSON integer"
			},
			"request_id": "req_test"
		}`))
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	_, err := New("").NoteGet("1e3")
	if err == nil {
		t.Fatal("HTTP 200 success=false must return an error")
	}
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("error type = %T, want *RequestError", err)
	}
	if requestErr.Field != "parent_id" ||
		requestErr.Constraint != "non_negative_decimal_integer" ||
		requestErr.ExpectedType != "decimal string or JSON integer" ||
		requestErr.RequestID != "req_test" ||
		requestErr.Retryable {
		t.Fatalf("structured error = %+v", requestErr)
	}
}

func TestNoteIDsRemainStringsWhenReencoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"note": {
					"id": 1916020531058082912,
					"note_id": "1916020531058082912",
					"children_ids": ["1916020531058082913"],
					"parent_id": 1916020531058082911,
					"parent_note_id": "1916020531058082911",
					"tags": [],
					"topics": [],
					"attachments": []
				}
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	resp, err := New("").NoteGet("1916020531058082912")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	note := decoded["data"].(map[string]interface{})["note"].(map[string]interface{})
	if note["note_id"] != "1916020531058082912" ||
		note["parent_note_id"] != "1916020531058082911" ||
		note["children_ids"].([]interface{})[0] != "1916020531058082913" {
		t.Fatalf("string IDs changed after round trip: %s", encoded)
	}
}

func TestSaveNoteSendsCompatibilityAndIdempotencyFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/api/v1/resource/note/save" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"topic_id":          "book-space",
			"parent_id":         "1916020531058082912",
			"client_request_id": "req-123",
		} {
			if body[key] != want {
				t.Fatalf("%s = %#v, want %q", key, body[key], want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"note_id":"1916020531058082913"}}`))
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	_, err := New("").NoteSave(NoteSaveRequest{
		NoteType:        "plain_text",
		Content:         "child",
		TopicID:         "book-space",
		ParentID:        "1916020531058082912",
		ClientRequestID: "req-123",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeAPIHost(t *testing.T) {
	for input, want := range map[string]string{
		"http://example.test":              "http://example.test",
		"http://example.test/open":         "http://example.test",
		"http://example.test/open/api/v1":  "http://example.test",
		"http://example.test/open/api/v1/": "http://example.test",
	} {
		if got := normalizeAPIHost(input); got != want {
			t.Errorf("normalizeAPIHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNoteURLUsesMatchingWebEnvironment(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
		webURL string
		want   string
	}{
		{name: "production", apiURL: "https://openapi.biji.com", want: "https://www.biji.com/note/1912345678901234567"},
		{name: "test gateway", apiURL: "http://entree.dev.didatrip.com/open", want: "http://biji.dev.didatrip.com/note/1912345678901234567"},
		{name: "explicit web host", apiURL: "https://openapi.biji.com", webURL: "https://preview.example.com/", want: "https://preview.example.com/note/1912345678901234567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GETNOTE_API_URL", tt.apiURL)
			t.Setenv("GETNOTE_WEB_URL", tt.webURL)
			got := New("").NoteURL("1912345678901234567")
			if got != tt.want {
				t.Fatalf("NoteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestErrorFromRateLimitBodyPreservesFields(t *testing.T) {
	err := requestErrorFromBody([]byte(`{
		"success": false,
		"error": {
			"code": 10202,
			"message": "请求频率超限",
			"reason": "qps_exceeded",
			"retryable": true
		},
		"request_id": "req_rate"
	}`), http.StatusTooManyRequests)
	if err == nil ||
		err.Code != 10202 ||
		err.Reason != "qps_exceeded" ||
		!err.Retryable ||
		err.RequestID != "req_rate" ||
		err.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("rate-limit error = %+v", err)
	}
}

func TestSnowflakeIDsAlwaysMarshalAsStrings(t *testing.T) {
	var response NoteListResponse
	if err := decodeJSON([]byte(`{
		"success": true,
		"data": {
			"notes": [{"id": 1916020531058082912, "parent_id": "1916020531058082911"}],
			"next_cursor": 1916020531058082913
		}
	}`), &response); err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	for _, want := range []string{
		`"id":"1916020531058082912"`,
		`"parent_id":"1916020531058082911"`,
		`"next_cursor":"1916020531058082913"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("structured output %s does not contain %s", got, want)
		}
	}
}

func TestNoteSaveNormalizesNumericIDAndAddsEnvironmentURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"note_id":1916020531058082913}}`))
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	t.Setenv("GETNOTE_WEB_URL", "https://preview.example.com")
	resp, err := New("").NoteSave(NoteSaveRequest{NoteType: "plain_text", Content: "test"})
	if err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data = %#v", resp.Data)
	}
	if data["note_id"] != "1916020531058082913" ||
		data["note_url"] != "https://preview.example.com/note/1916020531058082913" {
		t.Fatalf("normalized save data = %#v", data)
	}
}

func TestSkill2KnowledgeRoutesMatchOpenAPIContract(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/open/api/v1/resource/knowledge/directories":
			if r.URL.Query().Get("topic_id") != "topic-alias" || r.URL.Query().Get("directory_id") != "1916020531058082912" {
				t.Fatalf("directory query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"directories":[],"resources":[],"total":0}}`))
		case "/open/api/v1/resource/knowledge/directory/create":
			var body KBDirectoryRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.TopicID != "topic-alias" || body.ParentID != "1916020531058082912" || body.Name != "项目资料" {
				t.Fatalf("directory body = %+v", body)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"1916020531058082913"}}`))
		case "/open/api/v1/resource/knowledge/blogger/follow":
			_, _ = w.Write([]byte(`{"success":true,"data":{"follow_id_str":"1916020531058082914"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	c := New("")
	if _, err := c.KBDirectoryList("topic-alias", "1916020531058082912"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.KBDirectoryCreate("topic-alias", "1916020531058082912", "项目资料"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.KBBloggerFollow("topic-alias", "https://www.douyin.com/user/example", "douyin"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /open/api/v1/resource/knowledge/directories",
		"POST /open/api/v1/resource/knowledge/directory/create",
		"POST /open/api/v1/resource/knowledge/blogger/follow",
	}
	if strings.Join(requests, "|") != strings.Join(want, "|") {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
}

func TestSkill2FirstClassNoteFieldsDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {"note": {
				"note_id": "1916020531058082912",
				"quick_note": "关键结论",
				"attachments": [{"type":"image","name":"图一","url":"https://example.com/a.png"}],
				"timeline": {
					"version": 2,
					"moments": [{"start_ms":0,"end_ms":1200,"text":"开场"}],
					"resources": [{"type":"audio","url":"https://example.com/a.mp3","action_time":0}]
				}
			}}
		}`))
	}))
	defer server.Close()

	t.Setenv("GETNOTE_API_URL", server.URL)
	resp, err := New("").NoteGet("1916020531058082912")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.Note.QuickNote != "关键结论" || resp.Data.Note.Timeline == nil || len(resp.Data.Note.Timeline.Moments) != 1 || len(resp.Data.Note.Attachments) != 1 {
		t.Fatalf("first-class fields = %+v", resp.Data.Note)
	}
}
