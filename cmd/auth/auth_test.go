package auth

import "testing"

func TestOAuthURLFollowsAPIOverride(t *testing.T) {
	tests := map[string]string{
		"http://example.test":              "http://example.test/open/api/v1/oauth/token",
		"http://example.test/open":         "http://example.test/open/api/v1/oauth/token",
		"http://example.test/open/api/v1":  "http://example.test/open/api/v1/oauth/token",
		"http://example.test/open/api/v1/": "http://example.test/open/api/v1/oauth/token",
	}
	for baseURL, want := range tests {
		t.Run(baseURL, func(t *testing.T) {
			t.Setenv("GETNOTE_API_URL", baseURL)
			if got := oauthURL("/oauth/token"); got != want {
				t.Fatalf("oauthURL() = %q, want %q", got, want)
			}
		})
	}
}
