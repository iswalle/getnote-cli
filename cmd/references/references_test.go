package references

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExperienceReferenceCommandUsesContainerIdentity(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if r.URL.Path != "/open/api/v1/resource/reference/list" || q.Get("source") != "dedao" || q.Get("query") != "课程" {
			t.Error("lost lookup filters")
		}
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["reference.resources.v1","pagination.v1"]},"data":{"resources":[{"course_type":"66","course_id":"9007199254740993","has_notes":true}],"applied_filters":{"source":"dedao","resource_type":"course","query":"课程","match_mode":"exact"},"pagination":{"mode":"cursor","page_size":20,"returned_count":1,"has_more":false}}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "json", "")
	root.AddCommand(NewReferencesCmd())
	root.SetArgs([]string{"references", "list", "--source", "dedao", "--resource-type", "course", "--query", "课程", "--all"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(out.String(), `"course_id": "9007199254740993"`) || !strings.Contains(out.String(), `"completed": true`) {
		t.Fatalf("lost course identity or completion: %s", out.String())
	}
}
