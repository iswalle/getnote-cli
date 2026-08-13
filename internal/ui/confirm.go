package ui

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ConfirmDestructive requires an explicit confirmation before a destructive or
// visibility-changing operation. Machine-readable calls must pass --yes so a
// prompt can never corrupt JSON output or block an agent indefinitely.
func ConfirmDestructive(cmd *cobra.Command, approved bool, prompt string) (bool, error) {
	if approved {
		return true, nil
	}
	format, _ := cmd.Root().PersistentFlags().GetString("output")
	if format == "json" {
		return false, fmt.Errorf("该操作需要明确确认；确认后重新执行并添加 --yes")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N] ", prompt)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return false, fmt.Errorf("未收到确认；如已确认，请重新执行并添加 --yes")
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
