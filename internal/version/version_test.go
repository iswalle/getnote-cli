package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"v1.5.2", "1.5.1", 1}, {"1.5.2", "v1.5.2", 0}, {"1.5.1", "1.5.2", -1},
		{"2.0.0", "1.99.99", 1}, {"1.5.2-rc.1", "1.5.2", -1}, {"1.5.2+build.2", "1.5.2+build.1", 0},
		{"1.5.2-rc.10", "1.5.2-rc.2", 1},
	}
	for _, test := range tests {
		got := Compare(test.left, test.right)
		if got < 0 {
			got = -1
		} else if got > 0 {
			got = 1
		}
		if got != test.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}
