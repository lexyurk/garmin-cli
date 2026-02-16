package main

import (
	"strings"
	"testing"
)

func TestRenderNotAuthenticatedString_Markdown(t *testing.T) {
	got := renderNotAuthenticatedString("markdown", "")
	if !strings.Contains(got, "## Authentication") {
		t.Fatalf("missing title: %q", got)
	}
	if !strings.Contains(got, "not authenticated") {
		t.Fatalf("missing status: %q", got)
	}
}

func TestRenderNotAuthenticatedString_JSON(t *testing.T) {
	got := renderNotAuthenticatedString("json", "")
	if !strings.Contains(got, "\"error\": \"not_authenticated\"") {
		t.Fatalf("missing error json: %q", got)
	}
	if !strings.Contains(got, "\"profile\": \"default\"") {
		t.Fatalf("missing profile json: %q", got)
	}
}

