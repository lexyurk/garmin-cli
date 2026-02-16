package timeutil

import (
	"testing"
	"time"
)

func TestResolveDates_DefaultToday(t *testing.T) {
	now := time.Date(2026, 2, 16, 13, 0, 0, 0, time.Local)
	got, err := ResolveDates(RangeOptions{}, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0] != "2026-02-16" {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestResolveDates_Date(t *testing.T) {
	now := time.Date(2026, 2, 16, 13, 0, 0, 0, time.Local)
	got, err := ResolveDates(RangeOptions{Date: "2026-02-10"}, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0] != "2026-02-10" {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestResolveDates_Days(t *testing.T) {
	now := time.Date(2026, 2, 16, 13, 0, 0, 0, time.Local)
	got, err := ResolveDates(RangeOptions{Days: 3}, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"2026-02-14", "2026-02-15", "2026-02-16"}
	if len(got) != len(want) {
		t.Fatalf("unexpected: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected: %#v", got)
		}
	}
}

func TestResolveDates_FromTo(t *testing.T) {
	now := time.Date(2026, 2, 16, 13, 0, 0, 0, time.Local)
	got, err := ResolveDates(RangeOptions{From: "2026-02-01", To: "2026-02-03"}, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"2026-02-01", "2026-02-02", "2026-02-03"}
	if len(got) != len(want) {
		t.Fatalf("unexpected: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected: %#v", got)
		}
	}
}

func TestResolveDates_MutualExclusion(t *testing.T) {
	now := time.Date(2026, 2, 16, 13, 0, 0, 0, time.Local)
	_, err := ResolveDates(RangeOptions{Date: "2026-02-10", Days: 7}, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

