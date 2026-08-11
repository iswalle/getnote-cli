package capabilities

import (
	"bytes"
	"encoding/json"
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
	if got.ContractVersion != "2.0" {
		t.Fatalf("contract_version = %q, want 2.0", got.ContractVersion)
	}
	if len(got.Commands["notes"]) == 0 || len(got.Commands["knowledge_base"]) == 0 {
		t.Fatalf("missing command groups: %#v", got.Commands)
	}
	if got.Upgrade["check"] != "getnote update --check" || got.Upgrade["cli"] != "getnote update" {
		t.Fatalf("missing upgrade guidance: %#v", got.Upgrade)
	}
}
