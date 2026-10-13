package discoveryflags

import (
	"fmt"
	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/spf13/cobra"
)

func BindNotes(cmd *cobra.Command, o *client.DiscoveryOptions, topic bool) {
	f := cmd.Flags()
	f.StringVar(&o.NoteType, "note-type", "", "Exact note type, e.g. ref")
	f.StringVar(&o.Source, "source", "", "Exact source, e.g. dedao")
	f.StringVar(&o.ReferenceType, "reference-type", "", "Reference resource type: ebook, course_article, audio_book, unknown")
	f.StringVar(&o.ReferenceID, "reference-id", "", "Source resource ID; requires --source and --reference-type")
	f.StringVar(&o.ArticleURL, "article-url", "", "Exact article URL, including query and fragment")
	if topic {
		f.StringVar(&o.TopicID, "topic-id", "", "Limit to an authorized knowledge base")
	}
	f.IntVar(&o.PageSize, "page-size", 0, "Server page size (1..100, discovery default 20)")
	f.StringVar(&o.CreatedFrom, "created-from", "", "Created at or after RFC3339 time (inclusive)")
	f.StringVar(&o.CreatedTo, "created-to", "", "Created before RFC3339 time (exclusive)")
	f.StringVar(&o.EditedFrom, "edited-from", "", "Edited at or after RFC3339 time (inclusive)")
	f.StringVar(&o.EditedTo, "edited-to", "", "Edited before RFC3339 time (exclusive)")
	f.StringVar(&o.RecordUpdatedFrom, "record-updated-from", "", "Record updated at or after RFC3339 time; excludes relation changes")
	f.StringVar(&o.RecordUpdatedTo, "record-updated-to", "", "Record updated before RFC3339 time (exclusive)")
}
func ResolveNotes(cmd *cobra.Command, o client.DiscoveryOptions, limit int) (client.DiscoveryOptions, error) {
	if cmd.Flags().Changed("page-size") && cmd.Flags().Changed("limit") {
		return o, fmt.Errorf("use either --page-size or --limit")
	}
	if cmd.Flags().Changed("page-size") && o.PageSize < 1 {
		return o, fmt.Errorf("--page-size must be 1..100")
	}
	if cmd.Flags().Changed("limit") {
		if limit < 1 || limit > 100 {
			return o, fmt.Errorf("--limit must be 1..100")
		}
		if o.Active() {
			o.PageSize = limit
		}
	}
	if cmd.Flags().Changed("page") && o.Page < 1 {
		return o, fmt.Errorf("--page must be positive")
	}
	return o, nil
}
func BindLookup(cmd *cobra.Command, o *client.KnowledgeLookupOptions, all *bool) {
	f := cmd.Flags()
	f.StringVar(&o.Query, "query", "", "Match knowledge-base name or authoritative book title")
	f.StringVar(&o.Author, "author", "", "BOOKSPACE author keyword")
	f.StringVar(&o.BookID, "book-id", "", "Exact BOOKSPACE book ID")
	f.IntVar(&o.PageSize, "page-size", 0, "Server page size (1..100, discovery default 20)")
	f.BoolVar(all, "all", false, "Traverse all matching pages; no snapshot guarantee")
}
func LookupRequested(cmd *cobra.Command) bool {
	for _, key := range []string{"query", "author", "book-id", "page-size", "all"} {
		if cmd.Flags().Changed(key) {
			return true
		}
	}
	return false
}
func ValidateLookupFlags(cmd *cobra.Command, o client.KnowledgeLookupOptions) error {
	if cmd.Flags().Changed("page-size") && o.PageSize < 1 {
		return fmt.Errorf("--page-size must be 1..100")
	}
	if cmd.Flags().Changed("page") && o.Page < 1 {
		return fmt.Errorf("--page must be positive")
	}
	return nil
}
