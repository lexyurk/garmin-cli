// Package records reads Garmin Connect personal records (PRs).
package records

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

// recordLabels maps the well-known running PR type ids to labels. The mapping
// is community-inferred and intentionally conservative; unknown ids fall back
// to "type N". Use --format json for the full, unmapped payload.
var recordLabels = map[int]string{
	1: "1 km",
	2: "1 mile",
	3: "5 km",
	4: "10 km",
	5: "longest run",
}

type Record struct {
	TypeID       int     `json:"type_id"`
	Label        string  `json:"label,omitempty"`
	Value        float64 `json:"value,omitempty"`
	ActivityID   int64   `json:"activity_id,omitempty"`
	ActivityName string  `json:"activity_name,omitempty"`
}

type recordRaw struct {
	TypeID       int     `json:"typeId"`
	Value        float64 `json:"value"`
	ActivityID   int64   `json:"activityId"`
	ActivityName string  `json:"activityName"`
}

func recordPath(displayName string) string {
	return "/personalrecord-service/personalrecord/prs/" + url.PathEscape(displayName)
}

// List returns the user's personal records.
func List(ctx context.Context, c *client.Client, displayName string) ([]Record, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("display name is required")
	}
	var raw []recordRaw
	if err := c.GetJSON(ctx, recordPath(displayName), nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(raw))
	for _, r := range raw {
		label := recordLabels[r.TypeID]
		if label == "" {
			label = fmt.Sprintf("type %d", r.TypeID)
		}
		out = append(out, Record{
			TypeID:       r.TypeID,
			Label:        label,
			Value:        r.Value,
			ActivityID:   r.ActivityID,
			ActivityName: r.ActivityName,
		})
	}
	return out, nil
}

// ListRaw returns the raw PR payloads (for full-fidelity JSON output).
func ListRaw(ctx context.Context, c *client.Client, displayName string) ([]map[string]any, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("display name is required")
	}
	var raw []map[string]any
	if err := c.GetJSON(ctx, recordPath(displayName), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
