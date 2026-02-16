package auth

import "testing"

func TestExtractCSRF(t *testing.T) {
	html := `<html><body><input type="hidden" name="_csrf" value="csrf123"/></body></html>`
	got, err := extractCSRF(html)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "csrf123" {
		t.Fatalf("unexpected csrf: %q", got)
	}
}

func TestExtractTitle(t *testing.T) {
	html := `<html><head><title>My MFA Page</title></head></html>`
	if got := extractTitle(html); got != "My MFA Page" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestExtractTicket(t *testing.T) {
	html := `<html><body>... "https://sso.garmin.com/sso/embed?ticket=abcDEF123" ...</body></html>`
	got, err := extractTicket(html)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "abcDEF123" {
		t.Fatalf("unexpected ticket: %q", got)
	}
}
