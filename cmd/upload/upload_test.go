package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type uploadTransport func(*http.Request) (*http.Response, error)

func (f uploadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUploadCommandUsesOnlyOSSToken(t *testing.T) {
	t.Setenv("GETNOTE_API_KEY", "")
	t.Setenv("GETNOTE_CLIENT_ID", "")
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	calls := 0
	http.DefaultTransport = uploadTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "PUT" || r.URL.Host != "test.oss-cn-beijing.aliyuncs.com" {
			t.Fatal("unexpected destination")
		}
		for _, h := range []string{"Authorization", "Cookie", "X-Client-ID"} {
			if r.Header.Get(h) != "" {
				t.Fatalf("unexpected %s", h)
			}
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != "hello" {
			t.Fatal("wrong file bytes")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	token := `{"put_sign_url":"https://test.oss-cn-beijing.aliyuncs.com/object","put_content_type":"text/plain","get_url":"https://example.test/object"}`
	for _, input := range []string{token, `{"success":true,"data":` + token + `}`} {
		command := NewUploadCmd()
		command.SetArgs([]string{path, "--max-size-bytes", "5"})
		command.SetIn(strings.NewReader(input))
		var out bytes.Buffer
		command.SetOut(&out)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result["stage"] != "oss_uploaded" || result["file_type"] != "TXT" {
			t.Fatal("wrong result")
		}
		if strings.Contains(out.String(), "put_sign_url") {
			t.Fatal("token leaked")
		}
	}
	if calls != 2 {
		t.Fatal("expected two direct OSS requests only")
	}
}
