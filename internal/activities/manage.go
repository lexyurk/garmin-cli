package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type ActivityType struct {
	TypeID       int    `json:"typeId"`
	TypeKey      string `json:"typeKey"`
	ParentTypeID int    `json:"parentTypeId"`
}

// GetActivityTypes returns the catalog of activity types (typeKey -> ids).
func GetActivityTypes(ctx context.Context, c *client.Client) ([]ActivityType, error) {
	var out []ActivityType
	if err := c.GetJSON(ctx, "/activity-service/activity/activityTypes", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResolveType finds an activity type by its key (e.g. "running").
func ResolveType(types []ActivityType, key string) (ActivityType, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, t := range types {
		if strings.ToLower(t.TypeKey) == key {
			return t, nil
		}
	}
	return ActivityType{}, fmt.Errorf("unknown activity type %q", key)
}

// UpdateOptions describes editable fields of an activity. Nil fields are left unchanged.
type UpdateOptions struct {
	Name        *string
	Description *string
	Type        *ActivityType
}

// Update edits an activity's name, description, and/or type.
func Update(ctx context.Context, c *client.Client, activityID int64, opts UpdateOptions) error {
	body := map[string]any{"activityId": activityID}
	if opts.Name != nil {
		body["activityName"] = *opts.Name
	}
	if opts.Description != nil {
		body["description"] = *opts.Description
	}
	if opts.Type != nil {
		body["activityTypeDTO"] = map[string]any{
			"typeId":       opts.Type.TypeID,
			"typeKey":      opts.Type.TypeKey,
			"parentTypeId": opts.Type.ParentTypeID,
		}
	}
	if len(body) == 1 {
		return fmt.Errorf("nothing to update")
	}
	return c.PutJSON(ctx, fmt.Sprintf("/activity-service/activity/%d", activityID), nil, body, nil)
}

// Delete permanently removes an activity.
func Delete(ctx context.Context, c *client.Client, activityID int64) error {
	return c.Delete(ctx, fmt.Sprintf("/activity-service/activity/%d", activityID), nil)
}
