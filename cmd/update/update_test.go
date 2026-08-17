package update

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIsNPMManagedBinary(t *testing.T) {
	if !isNPMManagedBinary("/usr/local/lib/node_modules/@getnote/cli/bin/getnote") {
		t.Fatal("npm path not detected")
	}
	if isNPMManagedBinary("/usr/local/bin/getnote") {
		t.Fatal("standalone path detected as npm")
	}
}

func TestVerifyReleaseChecksum(t *testing.T) {
	content := []byte("archive")
	asset := "getnote-cli_1.0.0_linux_amd64.tar.gz"
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			fmt.Fprintf(w, "%s  %s\n", hash, asset)
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()
	archive := filepath.Join(t.TempDir(), asset)
	if err := os.WriteFile(archive, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyReleaseChecksum(server.URL+"/"+asset, asset, archive); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyReleaseChecksum(server.URL+"/"+asset, asset, archive); err == nil {
		t.Fatal("tampered archive must fail")
	}
}
