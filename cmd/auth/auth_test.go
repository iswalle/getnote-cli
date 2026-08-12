package auth

import "testing"

func TestResolveOAuthClientID(t *testing.T) {
	t.Run("environment override", func(t *testing.T) {
		t.Setenv("GETNOTE_CLIENT_ID", "cli_test_override")
		if got := resolveOAuthClientID(); got != "cli_test_override" {
			t.Fatalf("resolveOAuthClientID() = %q", got)
		}
	})
	t.Run("test environment default", func(t *testing.T) {
		if got := resolveOAuthClientIDForURL("http://entree.dev.didatrip.com/open", ""); got != testOAuthClientID {
			t.Fatalf("resolveOAuthClientID() = %q, want %q", got, testOAuthClientID)
		}
	})
	t.Run("production environment default", func(t *testing.T) {
		if got := resolveOAuthClientIDForURL("https://openapi.biji.com", ""); got != productionOAuthClientID {
			t.Fatalf("resolveOAuthClientID() = %q, want %q", got, productionOAuthClientID)
		}
	})
	t.Run("saved authorization wins", func(t *testing.T) {
		if got := resolveOAuthClientIDForURL("https://openapi.biji.com", "cli_saved"); got != "cli_saved" {
			t.Fatalf("resolveOAuthClientID() = %q, want cli_saved", got)
		}
	})
}

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
