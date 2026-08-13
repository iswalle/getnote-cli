package search

import (
	"context"
	"errors"
	"testing"
)

func TestIsSearchTimeout(t *testing.T) {
	t.Parallel()
	if !isSearchTimeout(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded must be recognized as timeout")
	}
	if isSearchTimeout(errors.New("search failed")) {
		t.Fatal("ordinary error must not be recognized as timeout")
	}
}
