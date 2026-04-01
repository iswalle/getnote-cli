// Package version holds the CLI version, injected at build time via ldflags.
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Version is set at build time: -ldflags "-X github.com/iswalle/getnote-cli/internal/version.Version=v0.1.0"
var Version = "dev"

const latestReleaseURL = "https://api.github.com/repos/iswalle/getnote-cli/releases/latest"

// LatestRelease fetches the latest release tag from GitHub.
// Returns empty string on any error (non-blocking).
func LatestRelease() string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}
	return result.TagName
}

// CheckUpdate compares current version to latest and returns an upgrade hint.
// Returns empty string if up-to-date or if check fails.
func CheckUpdate() string {
	if Version == "dev" {
		return ""
	}
	latest := LatestRelease()
	if latest == "" || latest == Version {
		return ""
	}
	// simple semver: if latest != current, suggest upgrade
	return fmt.Sprintf("A new version is available: %s → %s\nUpgrade: npm install -g @getnote/cli\n         or download from https://github.com/iswalle/getnote-cli/releases", Version, latest)
}

// String returns a formatted version string.
func String() string {
	return strings.TrimPrefix(Version, "v")
}
