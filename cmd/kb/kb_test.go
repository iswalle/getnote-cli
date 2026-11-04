package kb

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAddResult(t *testing.T) {
	cases := []struct {
		name       string
		data       interface{}
		requested  int
		wantAdded  int
		wantFailed []string
	}{
		{"all added", map[string]interface{}{"success_count": json.Number("2"), "failed_note_ids": []interface{}{}}, 2, 2, nil},
		{"rejected note", map[string]interface{}{"success_count": json.Number("0"), "failed_note_ids": []interface{}{"1925199248757135056"}}, 1, 0, []string{"1925199248757135056"}},
		{"partial", map[string]interface{}{"success_count": float64(1), "failed_note_ids": []interface{}{"2"}}, 2, 1, []string{"2"}},
		{"no count from older service", map[string]interface{}{"failed_note_ids": []interface{}{"2"}}, 3, 2, []string{"2"}},
		{"no data", nil, 2, 2, nil},
	}
	for _, tc := range cases {
		added, failed := addResult(tc.data, tc.requested)
		if added != tc.wantAdded || !reflect.DeepEqual(failed, tc.wantFailed) {
			t.Errorf("%s: got (%d, %v), want (%d, %v)", tc.name, added, failed, tc.wantAdded, tc.wantFailed)
		}
	}
}

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
