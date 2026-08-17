package capabilities

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

type response struct {
	Success         bool                     `json:"success"`
	CLIVersion      string                   `json:"cli_version"`
	ContractVersion string                   `json:"contract_version"`
	Architecture    string                   `json:"architecture"`
	Commands        map[string][]string      `json:"commands"`
	CommandAliases  map[string]string        `json:"command_aliases"`
	NoteTypes       []string                 `json:"note_types"`
	Platforms       []platform.Info          `json:"platforms"`
	Install         map[string]string        `json:"install"`
	Upgrade         map[string]string        `json:"upgrade"`
	ResultContracts map[string][]string      `json:"result_contracts"`
	CommandResults  map[string]commandResult `json:"command_results"`
	Guarantees      guarantees               `json:"guarantees"`
}

// commandResult is the stable, machine-readable result contract for one CLI
// command.  Skills use this rather than inferring success from prose or an
// HTTP status. Fields use dotted paths relative to the JSON response root.
// A command with no stable data payload explicitly declares that fact.
type commandResult struct {
	SuccessFields []string `json:"success_fields"`
	PendingFields []string `json:"pending_fields,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

type guarantees struct {
	IDsAsStrings             bool              `json:"ids_as_strings"`
	StructuredBusinessErrors bool              `json:"structured_business_errors"`
	FinalAsyncSaveResult     bool              `json:"final_async_save_result"`
	EnvironmentNoteURL       bool              `json:"environment_note_url"`
	ImageFormatValidation    bool              `json:"image_format_validation"`
	SafeLongInput            []string          `json:"safe_long_input"`
	KnowledgeScopes          []string          `json:"knowledge_scopes"`
	KnowledgeFeatures        []string          `json:"knowledge_features"`
	NoteDetailViews          []string          `json:"note_detail_views"`
	Limits                   map[string]int    `json:"limits"`
	ConfirmationFlags        map[string]string `json:"confirmation_flags"`
}

func commandResults() map[string]commandResult {
	return map[string]commandResult{
		"auth login": {
			SuccessFields: []string{"browser authorization completed", "local credential saved"},
			Notes:         "Interactive command. It succeeds only after the browser authorization is confirmed; never expose the credential.",
		},
		"auth status": {
			SuccessFields: []string{"authenticated=true|false", "credential_source?", "masked_credential?"},
			Notes:         "Human-readable status only. Any credential is masked; do not parse or display a full key.",
		},
		"auth logout": {
			SuccessFields: []string{"local credential removed"},
			Notes:         "Local-only action. It does not revoke server-side grants.",
		},
		"doctor": {
			SuccessFields: []string{"success", "diagnostics_completed=true", "ready", "status", "summary", "schema_version", "mode", "cli_version", "checks[].name", "checks[].ok", "checks[].code", "checks[].required", "checks[].severity", "issues[]", "next_actions[]", "update", "integrations[]", "platforms[]"},
			Notes:         "Use ready=true as the connection readiness decision. diagnostics_completed only means the diagnostic run finished. Follow blocking issues first, then execute next_actions only with the required user confirmation. platforms[] is legacy detection-only data; integrations[] reports Skill state.",
		},
		"capabilities": {
			SuccessFields: []string{"success=true", "contract_version", "commands", "command_aliases", "command_results", "guarantees"},
			Notes:         "This is the authoritative command and result contract for agents.",
		},
		"setup": {
			SuccessFields: []string{"success=true", "targets[]", "installed_skills", "authenticated", "next"},
			Notes:         "A missing supported target is not an account-authentication failure.",
		},
		"version": {
			SuccessFields: []string{"version text"},
			Notes:         "Human-readable version command; use capabilities for machine-readable contract discovery.",
		},
		"update": {
			SuccessFields: []string{"current version", "latest version or update result"},
			Notes:         "Interactive installer output. Run doctor after an update before claiming the CLI is ready.",
		},
		"quota": {
			SuccessFields: []string{"success=true", "data.read.daily", "data.read.monthly", "data.write.daily", "data.write.monthly", "data.write_note.daily", "data.write_note.monthly"},
		},
		"save": {
			SuccessFields: []string{"success=true", "data.note.note_id", "data.note.title", "data.note.note_url"},
			PendingFields: []string{"data.task_id", "data.status"},
			Notes:         "A link or image is not complete until the final note fields are present. On pending or timeout, query task with the same task_id; do not submit a second save.",
		},
		"task": {
			SuccessFields: []string{"success=true", "data.task_id", "data.status", "data.note_id?", "data.msg?", "data.error_msg?"},
			PendingFields: []string{"data.status=pending|processing"},
			Notes:         "Only done|success with a non-empty note_id is a completed save. failed is terminal.",
		},
		"notes": {
			SuccessFields: []string{"success=true", "data.notes[].note_id", "data.notes[].title", "data.notes[].note_url", "data.has_more", "data.cursor", "data.total"},
		},
		"note": {
			SuccessFields: []string{"success=true", "data.note.note_id", "data.note.title", "data.note.note_url", "data.note.note_type", "data.note.content?"},
		},
		"note original": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.original"},
		},
		"note transcript": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.transcript"},
		},
		"note attachments": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.attachments[]"},
		},
		"note timeline": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.timeline"},
		},
		"note quick-note": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.quick_note"},
		},
		"note todos": {
			SuccessFields: []string{"success=true", "data.note_id", "data.title", "data.meeting_todos[]"},
			Notes:         "Todos are rule-parsed from an explicit meeting-summary section; preserve each item's source and do not claim they are upstream-native tasks.",
		},
		"note update": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "A content or tag replacement requires --yes. Re-read note afterwards when the caller needs the final title, content or tags.",
		},
		"note delete": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Requires --yes. Success means the note moved to the recycle bin, not irreversible deletion.",
		},
		"note share": {
			SuccessFields: []string{"success=true", "data.note_id", "data.share_id", "data.share_url"},
			Notes:         "Requires --yes. share_url is public; do not create it without explicit user confirmation.",
		},
		"search": {
			SuccessFields: []string{"success=true", "data.results[].note_id?", "data.results[].title", "data.results[].note_url?", "data.results[].content?", "data.results[].score"},
			Notes:         "No results is a successful empty results array, not a search failure. On timeout, retry later or narrow the query / knowledge base; do not report an empty result.",
		},
		"kbs": {
			SuccessFields: []string{"success=true", "data.topics[].topic_id", "data.topics[].name", "data.topics[].scope", "data.topics[].stats", "data.has_more", "data.total"},
		},
		"kbs-sub": {
			SuccessFields: []string{"success=true", "data.topics[].topic_id", "data.topics[].name", "data.has_more", "data.total"},
		},
		"kb": {
			SuccessFields: []string{"success=true", "data.notes[].note_id", "data.notes[].title", "data.notes[].note_type", "data.has_more", "data.total"},
		},
		"kb create": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Only personal knowledge-base creation is supported. Use the returned object as-is; do not invent a topic_id if the upstream response omits it.",
		},
		"kb add": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "At most 20 note IDs. Re-read the directory or knowledge base when an itemized final state is needed.",
		},
		"kb remove": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Requires --yes. At most 20 note IDs.",
		},
		"kb directories": {
			SuccessFields: []string{"success=true", "data.current_directory?", "data.directories[].id", "data.directories[].name", "data.resources[]", "data.total"},
		},
		"kb directory-create": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "The name may be positional or passed with --name. Use returned IDs only. If an exact directory object is needed, re-read kb directories.",
		},
		"kb directory-update": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Use --name and/or --parent-id. Re-read kb directories when confirmation needs the final parent or name.",
		},
		"kb directory-delete": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Requires --yes and the service only deletes an empty directory. reason=knowledge_directory_not_empty means move its contents or delete child folders first; it is not retryable.",
		},
		"kb bloggers": {
			SuccessFields: []string{"success=true", "data.bloggers[].follow_id_str", "data.bloggers[].account_name", "data.bloggers[].platform", "data.has_more", "data.total"},
		},
		"kb blogger-follow": {
			SuccessFields: []string{"success=true", "data.follow_id_str", "data.url", "data.platform", "data.type", "data.created_at"},
		},
		"kb blogger-contents": {
			SuccessFields: []string{"success=true", "data.contents[].post_id_alias", "data.contents[].post_title", "data.contents[].post_summary?", "data.contents[].post_publish_time", "data.has_more", "data.total"},
		},
		"kb blogger-content": {
			SuccessFields: []string{"success=true", "data.post_id_alias", "data.post_title", "data.post_summary?", "data.post_media_text?", "data.post_url?", "data.post_publish_time"},
		},
		"kb lives": {
			SuccessFields: []string{"success=true", "data.lives[].live_id", "data.lives[].name", "data.lives[].status", "data.has_more", "data.total"},
		},
		"kb live": {
			SuccessFields: []string{"success=true", "data.post_title", "data.post_summary?", "data.post_media_text?", "data.post_publish_time"},
		},
		"kb live-follow": {
			SuccessFields: []string{"success=true", "data.follow_id_str", "data.url", "data.platform", "data.type", "data.created_at"},
		},
		"tag list": {
			SuccessFields: []string{"success=true", "data.note_id", "data.tags[].id", "data.tags[].name", "data.tags[].type"},
		},
		"tag add": {
			SuccessFields: []string{"success=true", "data.note_id", "data.tags[].id", "data.tags[].name", "data.tags[].type"},
		},
		"tag remove": {
			SuccessFields: []string{"success=true", "data?"},
			Notes:         "Re-read tag list when the remaining tag set must be shown.",
		},
	}
}

func currentResponse() response {
	return response{
		Success:         true,
		CLIVersion:      version.String(),
		ContractVersion: "2.1",
		Architecture:    "Skill navigates intent; CLI performs deterministic operations",
		Commands: map[string][]string{
			"connection":     {"doctor", "capabilities", "auth", "auth login", "auth status", "auth logout", "setup"},
			"notes":          {"save", "task", "notes", "note", "note original", "note transcript", "note attachments", "note timeline", "note quick-note", "note todos", "note update", "note delete", "note share"},
			"search":         {"search"},
			"knowledge_base": {"kbs", "kbs-sub", "kb", "kb create", "kb add", "kb remove", "kb directories", "kb directory-create", "kb directory-update", "kb directory-delete", "kb bloggers", "kb blogger-follow", "kb blogger-contents", "kb blogger-content", "kb lives", "kb live", "kb live-follow"},
			"tags":           {"tag", "tag list", "tag add", "tag remove"},
			"account":        {"quota", "version", "update"},
		},
		CommandAliases: map[string]string{
			"gnote":      "getnote",
			"kb dir":     "kb directories",
			"kb dirs":    "kb directories",
			"kb mkdir":   "kb directory-create",
			"kb mvdir":   "kb directory-update",
			"kb rmdir":   "kb directory-delete",
			"note quick": "note quick-note",
		},
		NoteTypes: []string{"plain_text", "link", "img_text"},
		Platforms: platform.Detect(),
		Install: map[string]string{
			"simple":   "Choose a supported AI and install the GetNote skill",
			"terminal": "npx -y @getnote/cli@latest setup",
			"fallback": "npx -y @getnote/mcp",
		},
		Upgrade: map[string]string{
			"check": "getnote update --check",
			"cli":   "getnote update",
			"npm":   "npm install -g @getnote/cli@latest",
		},
		ResultContracts: map[string][]string{
			"common_success": {"success=true", "data"},
			"common_error":   {"success=false", "data=null", "error.code", "error.message", "error.reason", "error.retryable", "request_id?"},
			"save":           {"data.note.note_id", "data.note.title", "data.note.note_url"},
			"task":           {"data.task_id", "data.status", "data.note_id?", "data.msg?", "data.error_msg?"},
			"notes":          {"data.notes[]", "data.total", "data.has_more", "data.cursor"},
			"note":           {"data.note.note_id", "data.note.title", "data.note.note_url", "data.note.note_type"},
			"search":         {"data.results[]"},
			"knowledge":      {"data.topics[]|data.notes[]|data.directories[]|data.resources[]"},
			"tags":           {"data.note_id", "data.tags[]"},
		},
		CommandResults: commandResults(),
		Guarantees: guarantees{
			IDsAsStrings:             true,
			StructuredBusinessErrors: true,
			FinalAsyncSaveResult:     true,
			EnvironmentNoteURL:       true,
			ImageFormatValidation:    true,
			SafeLongInput:            []string{"--content-file", "--stdin"},
			KnowledgeScopes:          []string{"DEFAULT", "BOOKSPACE", "CUSTOMER", "TEAMSPACE"},
			KnowledgeFeatures:        []string{"directories", "add_to_directory", "douyin_blogger_subscription"},
			NoteDetailViews:          []string{"summary", "original", "transcript", "attachments", "timeline", "quick_note", "meeting_todos"},
			Limits: map[string]int{
				"search_results": 10,
				"kb_note_batch":  20,
			},
			ConfirmationFlags: map[string]string{
				"note update content_or_tags": "--yes",
				"note delete":                 "--yes",
				"note share":                  "--yes",
				"kb remove":                   "--yes",
				"kb directory-delete":         "--yes",
			},
		},
	}
}

// NewCapabilitiesCmd reports the stable execution surface exposed to an AI host.
func NewCapabilitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "查看 CLI 能力与可接入平台 / Show CLI capabilities",
		Example: `  getnote capabilities
  getnote capabilities -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := currentResponse()
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "GetNote CLI %s\n", data.CLIVersion)
			groups := make([]string, 0, len(data.Commands))
			for group := range data.Commands {
				groups = append(groups, group)
			}
			sort.Strings(groups)
			for _, group := range groups {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", group, strings.Join(data.Commands[group], ", "))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Upgrade: getnote update")
			fmt.Fprintln(cmd.OutOrStdout(), "Detected AI platforms:")
			for _, item := range data.Platforms {
				mark := "-"
				if item.Detected {
					mark = "✓"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", mark, item.Name)
			}
			return nil
		},
	}
}
