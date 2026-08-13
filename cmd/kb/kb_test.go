package kb

import "testing"

func TestValidateNoteBatch(t *testing.T) {
	twenty := make([]string, 20)
	if err := validateNoteBatch(twenty); err != nil {
		t.Fatalf("20 notes should be accepted: %v", err)
	}
	twentyOne := make([]string, 21)
	if err := validateNoteBatch(twentyOne); err == nil {
		t.Fatal("21 notes should be rejected before calling the API")
	}
}
