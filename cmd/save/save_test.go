package save

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveContentFromArgs(t *testing.T) {
	got, err := resolveContent(&cobra.Command{}, []string{"一段", "文字"}, "", false)
	if err != nil || got != "一段 文字" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestResolveContentFromStdin(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader("很长的笔记\n第二行"))
	got, err := resolveContent(command, nil, "", true)
	if err != nil || got != "很长的笔记\n第二行" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestResolveContentRejectsEmptyStdin(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader(" \n"))
	if _, err := resolveContent(command, nil, "", true); err == nil {
		t.Fatal("expected empty stdin to fail")
	}
}
