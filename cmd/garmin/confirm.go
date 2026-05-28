package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// confirmDestructive returns true if the user confirms a destructive action.
//
// When force is true, it returns true without prompting. Otherwise it prompts
// on an interactive TTY; on a non-interactive stdin it returns false (the
// caller should treat that as "refused, pass --force").
func confirmDestructive(cmd *cobra.Command, prompt string, force bool) bool {
	if force {
		return true
	}
	inf, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(inf.Fd())) {
		return false
	}
	_, _ = cmd.ErrOrStderr().Write([]byte(prompt + " [y/N]: "))
	line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
