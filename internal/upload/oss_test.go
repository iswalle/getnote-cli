package upload

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(p, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func validToken() Token {
	return Token{PutURL: "https://bucket.oss-cn-beijing.aliyuncs.com/file.txt?Signature=SECRET", GetURL: "https://bucket.oss-cn-beijing.aliyuncs.com/file.txt", ContentType: "text/plain"}
}

func TestFileUsesOnlyOSSHeaders(t *testing.T) {
	c := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "PUT" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Client-ID") != "" {
			t.Fatal("unexpected authentication or method")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" || r.ContentLength != 5 {
			t.Fatal("wrong file body")
		}
		if _, exists := r.Header["X-Oss-Callback"]; !exists {
			t.Fatal("empty signed callback header omitted")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	r, err := uploadFile(context.Background(), c, fixture(t), validToken(), 10)
	if err != nil || r.Stage != "oss_uploaded" || r.MD5 != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("unexpected result: %v %v", r, err)
	}
}
func TestFileRejectsUntrustedHostsAndOversize(t *testing.T) {
	for _, raw := range []string{"http://bucket.oss-cn-beijing.aliyuncs.com/a", "https://example.com/a", "https://bucket.oss-cn-beijing.aliyuncs.com.evil.example/a", "https://user:pass@bucket.oss-cn-beijing.aliyuncs.com/a"} {
		token := validToken()
		token.PutURL = raw
		if validateToken(token) == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := uploadFile(context.Background(), http.DefaultClient, fixture(t), validToken(), 4); err == nil {
		t.Fatal("accepted oversized file")
	}
}
func TestErrorsNeverExposeSignedURL(t *testing.T) {
	c := &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, errors.New("SECRET") })}
	_, err := uploadFile(context.Background(), c, fixture(t), validToken(), 10)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatal("credential exposed")
	}
}
func TestRedirectIsNotSuccess(t *testing.T) {
	c := &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 307, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	if _, err := uploadFile(context.Background(), c, fixture(t), validToken(), 10); err == nil {
		t.Fatal("redirect accepted")
	}
}
