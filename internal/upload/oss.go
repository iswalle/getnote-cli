// Package upload performs only local-file-to-OSS transfer. It never reads an
// OpenAPI credential and never submits a Knowledge ingestion request.
package upload

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Token struct {
	PutURL      string `json:"put_sign_url"`
	ContentType string `json:"put_content_type"`
	ContentMD5  string `json:"put_md5"`
	Callback    string `json:"put_callback"`
	GetURL      string `json:"get_url"`
}

type Result struct {
	Stage    string `json:"stage"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	Size     int64  `json:"size_bytes"`
	MD5      string `json:"md5"`
	URL      string `json:"url"`
}

func validateToken(t Token) error {
	u, err := url.Parse(t.PutURL)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Path == "" || u.Fragment != "" {
		return errors.New("invalid HTTPS OSS upload URL")
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".aliyuncs.com") || !strings.Contains(host, ".oss-") {
		return errors.New("upload destination must be an Alibaba Cloud OSS bucket")
	}
	get, err := url.Parse(t.GetURL)
	if err != nil || get.Scheme != "https" || get.Host == "" || get.User != nil || get.Fragment != "" {
		return errors.New("invalid uploaded-file URL")
	}
	if t.ContentType == "" {
		return errors.New("upload token has no content type")
	}
	return nil
}

func File(ctx context.Context, path string, token Token, maxBytes int64) (*Result, error) {
	client := &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return uploadFile(ctx, client, path, token, maxBytes)
}

func uploadFile(ctx context.Context, client *http.Client, path string, token Token, maxBytes int64) (*Result, error) {
	if err := validateToken(token); err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		return nil, errors.New("max-size-bytes must come from the file capabilities API")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxBytes {
		return nil, errors.New("file must be non-empty, regular, and within the advertised size limit")
	}
	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, token.PutURL, f)
	if err != nil {
		return nil, errors.New("could not create OSS upload request")
	}
	req.ContentLength = info.Size()
	req.Header.Set("Content-Type", token.ContentType)
	req.Header.Set("Content-MD5", token.ContentMD5)
	req.Header.Set("x-oss-callback", token.Callback)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("OSS transfer failed; check connectivity or obtain a fresh upload token")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OSS transfer failed (HTTP %d)", resp.StatusCode)
	}
	return &Result{Stage: "oss_uploaded", FileName: filepath.Base(path), FileType: strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")), Size: info.Size(), MD5: hex.EncodeToString(hash.Sum(nil)), URL: token.GetURL}, nil
}
