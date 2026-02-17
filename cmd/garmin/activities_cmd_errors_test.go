package main

import (
	"strings"
	"testing"
)

func TestActivitiesExport_RejectsUnsupportedType(t *testing.T) {
	opts := &globalOptions{}
	cmd := newActivitiesExportCmd(opts)
	cmd.SetArgs([]string{"123", "--type", "zip"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported --type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesGet_RejectsInvalidID(t *testing.T) {
	opts := &globalOptions{}
	cmd := newActivitiesGetCmd(opts)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid activity id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivitiesSplits_RejectsInvalidID(t *testing.T) {
	opts := &globalOptions{}
	cmd := newActivitiesSplitsCmd(opts)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid activity id") {
		t.Fatalf("unexpected error: %v", err)
	}
}
