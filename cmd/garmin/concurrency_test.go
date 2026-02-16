package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMapDatesConcurrently_ReturnsRootErrorNotContextCanceled(t *testing.T) {
	rootErr := errors.New("root error")
	dates := []string{"a", "b", "c"}

	_, err := mapDatesConcurrently(context.Background(), dates, 3, func(ctx context.Context, date string) (string, error) {
		if date == "a" {
			return "", rootErr
		}
		// Wait for cancellation and then return ctx.Err.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			return "ok", nil
		}
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, rootErr) {
		t.Fatalf("expected root error, got: %v", err)
	}
}
