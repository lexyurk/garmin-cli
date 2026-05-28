// Package profile fetches the authenticated user's Garmin Connect profile.
package profile

import (
	"context"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type Profile struct {
	ProfileID   int64  `json:"profile_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	UserName    string `json:"user_name,omitempty"`
	Location    string `json:"location,omitempty"`
}

type socialProfileRaw struct {
	ProfileID   int64  `json:"profileId"`
	DisplayName string `json:"displayName"`
	FullName    string `json:"fullName"`
	UserName    string `json:"userName"`
	Location    string `json:"location"`
}

// Get returns the authenticated user's social profile.
func Get(ctx context.Context, c *client.Client) (Profile, error) {
	var raw socialProfileRaw
	if err := c.GetJSON(ctx, "/userprofile-service/socialProfile", nil, &raw); err != nil {
		return Profile{}, err
	}
	return Profile{
		ProfileID:   raw.ProfileID,
		DisplayName: raw.DisplayName,
		FullName:    raw.FullName,
		UserName:    raw.UserName,
		Location:    raw.Location,
	}, nil
}

// UserProfilePK returns the profile id used as userProfilePk in gear-service calls.
func UserProfilePK(ctx context.Context, c *client.Client) (int64, error) {
	p, err := Get(ctx, c)
	if err != nil {
		return 0, err
	}
	return p.ProfileID, nil
}
