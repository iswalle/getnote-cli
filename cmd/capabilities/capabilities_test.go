package capabilities

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCapabilitiesExposeSkill2ContractAndUpgrade(t *testing.T) {
	command := NewCapabilitiesCmd()
	command.PersistentFlags().StringP("output", "o", "table", "")
	if err := command.PersistentFlags().Set("output", "json"); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	var got response
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode capabilities: %v\n%s", err, output.String())
	}
	if got.ContractVersion != "2.1" {
		t.Fatalf("contract_version = %q, want 2.1", got.ContractVersion)
	}
	if got.CommandResults["save"].SuccessFields == nil {
		t.Fatal("capabilities must expose save result fields")
	}
	if len(got.Commands["notes"]) == 0 || len(got.Commands["knowledge_base"]) == 0 {
		t.Fatalf("missing command groups: %#v", got.Commands)
	}
	if got.Upgrade["check"] != "getnote update --check" || got.Upgrade["cli"] != "getnote update" {
		t.Fatalf("missing upgrade guidance: %#v", got.Upgrade)
	}
	if got.CommandAliases["gnote"] != "getnote" || got.CommandAliases["kb dir"] != "kb directories" {
		t.Fatalf("missing compact command aliases: %#v", got.CommandAliases)
	}
	for _, key := range []string{"common_success", "common_error", "save", "task", "notes", "note", "search", "knowledge", "tags"} {
		if len(got.ResultContracts[key]) == 0 {
			t.Fatalf("missing result contract %s: %#v", key, got.ResultContracts)
		}
	}
	if strings.Join(got.ResultContracts["search"], ",") != "data.results[]" ||
		strings.Join(got.ResultContracts["tags"], ",") != "data.note_id,data.tags[]" {
		t.Fatalf("incorrect result paths: %#v", got.ResultContracts)
	}
}

func TestContractPublishesHistoricalSafetyGuarantees(t *testing.T) {
	data := currentResponse()
	if data.ContractVersion != "2.1" ||
		!data.Guarantees.IDsAsStrings ||
		!data.Guarantees.StructuredBusinessErrors ||
		!data.Guarantees.FinalAsyncSaveResult ||
		!data.Guarantees.EnvironmentNoteURL ||
		!data.Guarantees.ImageFormatValidation {
		t.Fatalf("missing execution guarantees: %+v", data.Guarantees)
	}
	if data.Guarantees.Limits["search_results"] != 10 ||
		data.Guarantees.Limits["kb_note_batch"] != 20 {
		t.Fatalf("historical limits = %+v", data.Guarantees.Limits)
	}
	if strings.Join(data.Guarantees.KnowledgeScopes, ",") != "DEFAULT,BOOKSPACE,CUSTOMER,TEAMSPACE" {
		t.Fatalf("knowledge scopes = %v", data.Guarantees.KnowledgeScopes)
	}
	if strings.Join(data.Guarantees.KnowledgeFeatures, ",") != "directories,add_to_directory,douyin_blogger_subscription" {
		t.Fatalf("knowledge features = %v", data.Guarantees.KnowledgeFeatures)
	}
	if strings.Join(data.Guarantees.NoteDetailViews, ",") != "summary,original,transcript,attachments,timeline,quick_note,meeting_todos" {
		t.Fatalf("note detail views = %v", data.Guarantees.NoteDetailViews)
	}
	for _, command := range []string{"note update content_or_tags", "note delete", "note share", "kb remove", "kb directory-delete"} {
		if data.Guarantees.ConfirmationFlags[command] != "--yes" {
			t.Fatalf("%s confirmation = %q", command, data.Guarantees.ConfirmationFlags[command])
		}
	}
}
