package training

import (
	"context"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type VO2Max struct {
	Running float64 `json:"running"`
	Cycling float64 `json:"cycling"`
}

func GetVO2Max(ctx context.Context, c *client.Client) (VO2Max, error) {
	var raw map[string]any
	if err := c.GetJSON(ctx, "/userprofile-service/userprofile/user-settings", nil, &raw); err != nil {
		return VO2Max{}, err
	}
	userData, _ := raw["userData"].(map[string]any)
	return VO2Max{
		Running: floatFromAny(userData["vo2MaxRunning"]),
		Cycling: floatFromAny(userData["vo2MaxCycling"]),
	}, nil
}

func floatFromAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

