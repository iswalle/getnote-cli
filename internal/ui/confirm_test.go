package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func confirmationCommand(input string, jsonOutput bool) *cobra.Command {
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	if jsonOutput {
		_ = root.PersistentFlags().Set("output", "json")
	}
	root.SetIn(strings.NewReader(input))
	root.SetOut(&bytes.Buffer{})
	return root
}

func TestConfirmDestructiveApprovedByFlag(t *testing.T) {
	cmd := confirmationCommand("", true)
	ok, err := ConfirmDestructive(cmd, true, "Delete?")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestConfirmDestructivePromptsHuman(t *testing.T) {
	cmd := confirmationCommand("yes\n", false)
	ok, err := ConfirmDestructive(cmd, false, "Delete?")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestConfirmDestructiveRequiresYesForJSON(t *testing.T) {
	cmd := confirmationCommand("yes\n", true)
	if _, err := ConfirmDestructive(cmd, false, "Delete?"); err == nil {
		t.Fatal("JSON operation without --yes must fail")
	}
}
