package references

import (
	"encoding/json"
	"fmt"
	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewReferencesCmd() *cobra.Command {
	root := &cobra.Command{Use: "references", Short: "按课程名或老师查找有笔记的课程 / Locate course containers"}
	var o client.ReferenceLookupOptions
	var all bool
	cmd := &cobra.Command{Use: "list", Short: "按课程名精确/前缀匹配，老师精确匹配；仅个人笔记", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("page-size") && o.PageSize < 1 {
			return fmt.Errorf("--page-size must be 1..100")
		}
		c := client.New("")
		var r *client.ReferenceListResponse
		var err error
		if all {
			r, err = c.AllReferences(o)
		} else {
			r, err = c.ReferenceList(o)
		}
		if err != nil {
			return err
		}
		format, _ := cmd.Root().PersistentFlags().GetString("output")
		if format == "json" {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(r)
		}
		cols := []ui.ColSpec{{Value: "Type", Width: 6}, {Value: "Course ID", Width: 22}, {Value: "Title", Width: 40}, {Value: "Author", Width: 24}}
		fmt.Fprint(cmd.OutOrStdout(), ui.PrintHeader(cols, "  "))
		for _, v := range r.Data.Resources {
			fmt.Fprint(cmd.OutOrStdout(), ui.PrintRow([]ui.ColSpec{{Value: v.CourseType, Width: 6}, {Value: v.CourseID, Width: 22}, {Value: v.Title, Width: 40}, {Value: v.Author, Width: 24}}, "  "))
		}
		if p := r.Data.Pagination; p != nil && p.HasMore {
			fmt.Fprintf(cmd.OutOrStdout(), "More candidates remain; use --cursor %s or --all, including after an empty page.\n", *p.NextCursor)
		}
		if r.Data.Partial || len(r.Data.Warnings) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Partial metadata; warnings: %v\n", r.Data.Warnings)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d course containers returned in this request/traversal; no snapshot guarantee.\n", len(r.Data.Resources))
		return nil
	}}
	cmd.Example = "  getnote references list --source dedao --resource-type course --query '课程名' --all -o json\n  getnote references list --source dedao --resource-type course --author '老师姓名' --match-mode exact -o json"
	f := cmd.Flags()
	f.StringVar(&o.Source, "source", "", "Source, currently dedao (required)")
	f.StringVar(&o.ResourceType, "resource-type", "", "Container type, currently course (required)")
	f.StringVar(&o.Query, "query", "", "Course name only; supply query or author")
	f.StringVar(&o.Author, "author", "", "Exact teacher name; combined with course query using AND")
	f.StringVar(&o.MatchMode, "match-mode", "exact", "Course name matching: exact or prefix")
	f.IntVar(&o.PageSize, "page-size", 20, "Candidates per page (1..100)")
	f.StringVar(&o.Cursor, "cursor", "", "Opaque continuation; preserve all lookup conditions")
	f.BoolVar(&all, "all", false, "Follow advancing cursors, including empty pages; no snapshot guarantee")
	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("resource-type")
	root.AddCommand(cmd)
	return root
}
