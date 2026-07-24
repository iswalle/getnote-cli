package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
