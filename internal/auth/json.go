package auth

import (
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/lexyurk/garmin-cli/internal/config"
)

func loadJSON[T any](path string) (T, error) {
	var zero T
	b, err := config.ReadFile(path, 10<<20) // 10MB hard cap
	if err != nil {
		return zero, err
	}
	if err := json.Unmarshal(b, &zero); err != nil {
		return zero, fmt.Errorf("parse %s: %w", path, err)
	}
	return zero, nil
}

func saveJSON(path string, v any, perm fs.FileMode) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return config.WriteFileAtomic(path, b, perm)
}

