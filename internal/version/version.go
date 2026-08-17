// Package version holds the CLI version, injected at build time via ldflags.
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	if latest == "" || Compare(latest, Version) <= 0 {
		return ""
	}
	return fmt.Sprintf("A new version is available: %s → %s\nUpgrade: npm install -g @getnote/cli\n         or download from https://github.com/iswalle/getnote-cli/releases", Version, latest)
}

// Compare compares two semantic versions. It returns -1, 0 or 1. A leading v
// and build metadata are ignored; prereleases sort before the corresponding release.
func Compare(left, right string) int {
	parse := func(raw string) ([]int, []string, bool) {
		raw = strings.TrimPrefix(strings.TrimSpace(raw), "v")
		raw = strings.SplitN(raw, "+", 2)[0]
		parts := strings.SplitN(raw, "-", 2)
		core := strings.Split(parts[0], ".")
		if len(core) != 3 {
			return nil, nil, false
		}
		numbers := make([]int, 3)
		for i, part := range core {
			value, err := strconv.Atoi(part)
			if err != nil || value < 0 {
				return nil, nil, false
			}
			numbers[i] = value
		}
		var pre []string
		if len(parts) == 2 {
			pre = strings.Split(parts[1], ".")
		}
		return numbers, pre, true
	}
	ln, lp, lok := parse(left)
	rn, rp, rok := parse(right)
	if !lok || !rok {
		return strings.Compare(strings.TrimPrefix(left, "v"), strings.TrimPrefix(right, "v"))
	}
	for i := range ln {
		if ln[i] < rn[i] {
			return -1
		}
		if ln[i] > rn[i] {
			return 1
		}
	}
	if len(lp) == 0 && len(rp) > 0 {
		return 1
	}
	if len(lp) > 0 && len(rp) == 0 {
		return -1
	}
	for i := 0; i < len(lp) || i < len(rp); i++ {
		if i >= len(lp) {
			return -1
		}
		if i >= len(rp) {
			return 1
		}
		li, le := strconv.Atoi(lp[i])
		ri, re := strconv.Atoi(rp[i])
		switch {
		case le == nil && re == nil && li != ri:
			if li < ri {
				return -1
			}
			return 1
		case le == nil && re != nil:
			return -1
		case le != nil && re == nil:
			return 1
		case lp[i] != rp[i]:
			return strings.Compare(lp[i], rp[i])
		}
	}
	return 0
}

// String returns a formatted version string.
func String() string {
	return strings.TrimPrefix(Version, "v")
}
