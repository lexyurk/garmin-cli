package main

import (
	"testing"
	"time"
)

func TestResolveActivitiesDateFilters_Days(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	after, before, err := resolveActivitiesDateFilters("", "", "", "", 7, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if after != "2026-02-10" || before != "2026-02-16" {
		t.Fatalf("unexpected range: after=%s before=%s", after, before)
	}
}

func TestResolveActivitiesDateFilters_Days1(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	after, before, err := resolveActivitiesDateFilters("", "", "", "", 1, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if after != "2026-02-16" || before != "2026-02-16" {
		t.Fatalf("unexpected range: after=%s before=%s", after, before)
	}
}

func TestResolveActivitiesDateFilters_Passthrough(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	after, before, err := resolveActivitiesDateFilters("2026-01-01", "2026-01-31", "", "", 0, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if after != "2026-01-01" || before != "2026-01-31" {
		t.Fatalf("unexpected passthrough: after=%s before=%s", after, before)
	}
}

func TestResolveActivitiesDateFilters_DaysConflictsWithAfterBefore(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	_, _, err := resolveActivitiesDateFilters("2026-01-01", "", "", "", 7, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveActivitiesDateFilters_FromTo(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	after, before, err := resolveActivitiesDateFilters("", "", "2026-01-01", "2026-01-31", 0, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if after != "2026-01-01" || before != "2026-01-31" {
		t.Fatalf("unexpected from/to mapping: after=%s before=%s", after, before)
	}
}

func TestResolveActivitiesDateFilters_AfterConflictsWithFrom(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	_, _, err := resolveActivitiesDateFilters("2026-01-01", "", "2026-01-02", "", 0, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveActivitiesDateFilters_BeforeConflictsWithTo(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	_, _, err := resolveActivitiesDateFilters("", "2026-01-31", "", "2026-01-30", 0, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveActivitiesDateFilters_DaysNegative(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.Local)

	_, _, err := resolveActivitiesDateFilters("", "", "", "", -1, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}
