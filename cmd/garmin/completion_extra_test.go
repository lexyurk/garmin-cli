package main

import (
	"bytes"
	"testing"
)

func TestCompletion_OtherShells(t *testing.T) {
	for _, shell := range []string{"zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			cmd := NewRootCmd("dev")
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"completion", shell})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if out.Len() == 0 {
				t.Fatalf("expected completion output, got empty")
			}
		})
	}
}
