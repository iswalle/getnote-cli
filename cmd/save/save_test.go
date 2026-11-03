package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveContentFromArgs(t *testing.T) {
	got, err := resolveContent(&cobra.Command{}, []string{"一段", "文字"}, "", false)
	if err != nil || got != "一段 文字" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestValidateIdempotencyKey(t *testing.T) {
	for _, key := range []string{"request-123", strings.Repeat("a", 128)} {
		if err := validateIdempotencyKey(key); err != nil {
			t.Fatalf("valid key %q rejected: %v", key, err)
		}
	}
	for _, key := range []string{"has space", "中文", strings.Repeat("a", 129)} {
		if err := validateIdempotencyKey(key); err == nil {
			t.Fatalf("invalid key %q accepted", key)
		}
	}
}

func TestIsImagePathAcceptsBareRelativePath(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.WriteFile("photo.png", []byte("not checked here"), 0600); err != nil {
		t.Fatal(err)
	}
	if !isImagePath("photo.png") {
		t.Fatal("bare image path should be detected")
	}
	if isImagePath("missing.png") || isImagePath("photo.txt") {
		t.Fatal("missing or unsupported image must not be detected")
	}
}

func TestValidateImageFormatRejectsExtensionMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.jpg")
	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0, 'I', 'H', 'D', 'R'}
	if err := os.WriteFile(path, pngHeader, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := validateImageFormat(path); err == nil {
		t.Fatal("mismatched image extension must fail")
	}
}

func TestExtractNoteIDPreservesJSONNumber(t *testing.T) {
	got := extractNoteID(map[string]interface{}{"note_id": json.Number("1916020531058082912")})
	if got != "1916020531058082912" {
		t.Fatalf("note ID = %q", got)
	}
}

func TestResolveContentFromStdin(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader("很长的笔记\n第二行"))
	got, err := resolveContent(command, nil, "", true)
	if err != nil || got != "很长的笔记\n第二行" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestResolveContentRejectsEmptyStdin(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader(" \n"))
	if _, err := resolveContent(command, nil, "", true); err == nil {
		t.Fatal("expected empty stdin to fail")
	}
}

func TestResolveContentRejectsWhitespaceAcrossInputSources(t *testing.T) {
	for _, input := range []string{"   ", "\t\r\n", "\u3000\u00a0"} {
		for _, source := range []string{"args", "stdin", "file"} {
			t.Run(source+input, func(t *testing.T) {
				c := &cobra.Command{}
				var args []string
				file := ""
				stdin := false
				switch source {
				case "args":
					args = []string{input}
				case "stdin":
					stdin = true
					c.SetIn(strings.NewReader(input))
				case "file":
					file = filepath.Join(t.TempDir(), "input.txt")
					if err := os.WriteFile(file, []byte(input), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := resolveContent(c, args, file, stdin); err == nil {
					t.Fatal("whitespace accepted")
				}
			})
		}
	}
	for _, input := range []string{"  正文 🌱  ", "\n第一行\n第二行\n"} {
		got, err := resolveContent(&cobra.Command{}, []string{input}, "", false)
		if err != nil || got != input {
			t.Fatalf("valid content changed: %q %v", got, err)
		}
	}
}

func TestResumeUploadedImageDoesNotMintOrUpload(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/open/api/v1/resource/note/save" {
			t.Errorf("unexpected upload/request %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["note_type"] != "img_text" || body["client_request_id"] != "note-original" {
			t.Errorf("changed operation: %+v", body)
		}
		images, _ := body["image_urls"].([]any)
		if len(images) != 1 || images[0] != "https://example.com/original.png" {
			t.Errorf("changed image: %+v", body)
		}
		fmt.Fprint(w, `{"success":true,"data":{}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	path := filepath.Join(t.TempDir(), "url.txt")
	os.WriteFile(path, []byte("https://example.com/original.png"), 0600)
	cmd := NewSaveCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--image-url-file", path, "--idempotency-key", "note-original"})
	// Missing result ID is deliberately uncertain, but it must never upload again.
	if err := cmd.Execute(); err == nil {
		t.Fatal("unconfirmed save reported success")
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestImageWorkflowPreservesDefaultSaveKeyAfterUpload(t *testing.T) {
	uploaded, saved := 0, 0
	var noteKey string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/api/v1/resource/image/config":
			fmt.Fprint(w, `{"success":true,"data":{}}`)
		case "/open/api/v1/resource/image/upload_token":
			fmt.Fprintf(w, `{"success":true,"data":{"host":%q,"object_key":"original.png","accessid":"fixture","policy":"fixture","signature":"fixture","access_url":"https://example.com/original.png"}}`, server.URL+"/oss")
		case "/oss":
			uploaded++
			fmt.Fprint(w, `{}`)
		case "/open/api/v1/resource/note/save":
			saved++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			noteKey, _ = body["client_request_id"].(string)
			w.WriteHeader(503)
			fmt.Fprint(w, `{"success":false,"error":{"code":50300,"reason":"unavailable","retryable":false},"request_id":"save-failed"}`)
		default:
			t.Error(r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	path := filepath.Join(t.TempDir(), "image.png")
	os.WriteFile(path, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0, 'I', 'H', 'D', 'R'}, 0600)
	cmd := NewSaveCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{path})
	err := cmd.Execute()
	var resumed *uploadedImageError
	if !errors.As(err, &resumed) || uploaded != 1 || saved != 1 || noteKey == "" || resumed.NoteKey != noteKey || resumed.URL != "https://example.com/original.png" {
		t.Fatalf("lost progress: upload=%d save=%d key=%q err=%v", uploaded, saved, noteKey, err)
	}
}
