package doctor

import "testing"

func TestTemporaryNpxPath(t *testing.T) {
	tests := map[string]bool{
		"/Users/test/.npm/_npx/abc/node_modules/@getnote/cli/bin/getnote": true,
		"/tmp/_npx/abc/getnote":                         true,
		"/opt/homebrew/bin/getnote":                     false,
		"C:/Users/test/AppData/Roaming/npm/getnote.cmd": false,
	}
	for path, want := range tests {
		if got := isTemporaryNpxPath(path); got != want {
			t.Fatalf("isTemporaryNpxPath(%q) = %v, want %v", path, got, want)
		}
	}
}
