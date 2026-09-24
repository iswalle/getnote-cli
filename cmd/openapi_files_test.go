package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadUploadMetadata(t *testing.T) {
	valid := map[string]any{"stage": "oss_uploaded", "file_name": "sample.txt", "file_type": "TXT", "size_bytes": 12, "md5": strings.Repeat("a", 32), "url": "https://example.test/sample.txt"}
	encode := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	if result, err := readUploadMetadata(strings.NewReader(encode(valid))); err != nil || result.FileName != "sample.txt" {
		t.Fatalf("valid metadata: %v", err)
	}
	for _, input := range []string{encode(valid) + `{}`, encode(valid) + `garbage`, strings.Repeat(" ", 65537), `null`, `{}`} {
		if _, err := readUploadMetadata(strings.NewReader(input)); err == nil {
			t.Fatal("accepted invalid or oversized JSON")
		}
	}
	for key, value := range map[string]any{"file_base64": "secret", "file_path": "/private/file", "stage": "UPLOADING", "size_bytes": 0, "file_name": "", "file_type": "", "md5": "bad", "url": "http://example.test/file"} {
		copy := make(map[string]any)
		for k, v := range valid {
			copy[k] = v
		}
		copy[key] = value
		if _, err := readUploadMetadata(strings.NewReader(encode(copy))); err == nil {
			t.Errorf("accepted invalid %s", key)
		}
	}
}
