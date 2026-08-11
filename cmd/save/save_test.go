package save

import (
	"encoding/json"
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
