// Package gear manages Garmin Connect gear (shoes, bikes, etc.).
package gear

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type Gear struct {
	UUID        string   `json:"uuid,omitempty"`
	GearPk      int64    `json:"gear_pk,omitempty"`
	Name        string   `json:"name,omitempty"`
	Make        string   `json:"make,omitempty"`
	Model       string   `json:"model,omitempty"`
	Type        string   `json:"type,omitempty"`
	Status      string   `json:"status,omitempty"`
	MaxMeters   float64  `json:"max_meters,omitempty"`
	DateBegin   string   `json:"date_begin,omitempty"`
	DateEnd     string   `json:"date_end,omitempty"`
	TotalMeters *float64 `json:"total_meters,omitempty"`
	Activities  *int     `json:"total_activities,omitempty"`
}

type Stats struct {
	TotalMeters     float64 `json:"total_meters"`
	TotalActivities int     `json:"total_activities"`
}

type gearRaw struct {
	UUID            string  `json:"uuid"`
	GearPk          int64   `json:"gearPk"`
	GearMakeName    string  `json:"gearMakeName"`
	GearModelName   string  `json:"gearModelName"`
	CustomMakeModel string  `json:"customMakeModel"`
	DisplayName     string  `json:"displayName"`
	GearTypeName    string  `json:"gearTypeName"`
	GearStatusName  string  `json:"gearStatusName"`
	MaximumMeters   float64 `json:"maximumMeters"`
	DateBegin       string  `json:"dateBegin"`
	DateEnd         string  `json:"dateEnd"`
}

type statsRaw struct {
	TotalDistance   float64 `json:"totalDistance"`
	TotalActivities int     `json:"totalActivities"`
}

func (g gearRaw) toGear() Gear {
	return Gear{
		UUID:      g.UUID,
		GearPk:    g.GearPk,
		Name:      gearName(g),
		Make:      g.GearMakeName,
		Model:     g.GearModelName,
		Type:      g.GearTypeName,
		Status:    g.GearStatusName,
		MaxMeters: g.MaximumMeters,
		DateBegin: g.DateBegin,
		DateEnd:   g.DateEnd,
	}
}

func gearName(g gearRaw) string {
	if s := strings.TrimSpace(g.DisplayName); s != "" {
		return s
	}
	if s := strings.TrimSpace(g.CustomMakeModel); s != "" {
		return s
	}
	mm := strings.TrimSpace(strings.TrimSpace(g.GearMakeName) + " " + strings.TrimSpace(g.GearModelName))
	return mm
}

// List returns all gear for the given user profile.
func List(ctx context.Context, c *client.Client, userProfilePk int64) ([]Gear, error) {
	q := url.Values{"userProfilePk": {strconv.FormatInt(userProfilePk, 10)}}
	var raw []gearRaw
	if err := c.GetJSON(ctx, "/gear-service/gear/filterGear", q, &raw); err != nil {
		return nil, err
	}
	out := make([]Gear, 0, len(raw))
	for _, g := range raw {
		out = append(out, g.toGear())
	}
	return out, nil
}

// GetStats returns cumulative distance/activity totals for a gear item.
func GetStats(ctx context.Context, c *client.Client, uuid string) (Stats, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return Stats{}, fmt.Errorf("gear uuid is required")
	}
	var raw statsRaw
	if err := c.GetJSON(ctx, "/gear-service/gear/stats/"+url.PathEscape(uuid), nil, &raw); err != nil {
		return Stats{}, err
	}
	return Stats{TotalMeters: raw.TotalDistance, TotalActivities: raw.TotalActivities}, nil
}

// Get returns a single gear item (with stats) by uuid.
func Get(ctx context.Context, c *client.Client, userProfilePk int64, uuid string) (Gear, error) {
	uuid = strings.TrimSpace(uuid)
	gears, err := List(ctx, c, userProfilePk)
	if err != nil {
		return Gear{}, err
	}
	for _, g := range gears {
		if strings.EqualFold(g.UUID, uuid) {
			st, err := GetStats(ctx, c, g.UUID)
			if err == nil {
				g.TotalMeters = &st.TotalMeters
				g.Activities = &st.TotalActivities
			}
			return g, nil
		}
	}
	return Gear{}, fmt.Errorf("gear %q not found", uuid)
}

// WithStats fetches per-gear cumulative stats and populates each gear in place.
func WithStats(ctx context.Context, c *client.Client, gears []Gear) []Gear {
	for i := range gears {
		st, err := GetStats(ctx, c, gears[i].UUID)
		if err != nil {
			continue
		}
		m := st.TotalMeters
		a := st.TotalActivities
		gears[i].TotalMeters = &m
		gears[i].Activities = &a
	}
	return gears
}

// CreateOptions describes a new gear item.
type CreateOptions struct {
	Type      string  // gear type name, e.g. "Shoes", "Bike" (default "Shoes")
	Name      string  // user-facing name (required)
	Make      string  // optional manufacturer
	Model     string  // optional model
	MaxMeters float64 // optional retirement threshold (0 = none)
	DateBegin string  // YYYY-MM-DD (default today)
}

// Create adds a new gear item.
//
// The gear-service write contract is not officially documented; the request
// body mirrors the gear object returned by reads. Garmin may reject unexpected
// fields, in which case the API error is surfaced verbatim.
func Create(ctx context.Context, c *client.Client, userProfilePk int64, opts CreateOptions) (Gear, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return Gear{}, fmt.Errorf("gear name is required")
	}
	typeName := strings.TrimSpace(opts.Type)
	if typeName == "" {
		typeName = "Shoes"
	}
	begin := strings.TrimSpace(opts.DateBegin)
	if begin == "" {
		begin = time.Now().In(time.Local).Format("2006-01-02")
	}

	body := map[string]any{
		"userProfilePk":   userProfilePk,
		"gearTypeName":    typeName,
		"gearMakeName":    strings.TrimSpace(opts.Make),
		"gearModelName":   strings.TrimSpace(opts.Model),
		"customMakeModel": name,
		"displayName":     name,
		"maximumMeters":   opts.MaxMeters,
		"dateBegin":       begin + "T00:00:00.0",
		"gearStatusName":  "active",
	}

	var out gearRaw
	if err := c.PostJSON(ctx, "/gear-service/gear", nil, body, &out); err != nil {
		return Gear{}, err
	}
	return out.toGear(), nil
}

// SetStatus retires ("retired") or restores ("active") a gear item.
//
// It re-PUTs the existing gear object with the status field changed, preserving
// all other fields. The gear-service write contract is reverse-engineered.
func SetStatus(ctx context.Context, c *client.Client, userProfilePk int64, uuid, status string) (Gear, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return Gear{}, fmt.Errorf("gear uuid is required")
	}

	var raw []map[string]any
	q := url.Values{"userProfilePk": {strconv.FormatInt(userProfilePk, 10)}}
	if err := c.GetJSON(ctx, "/gear-service/gear/filterGear", q, &raw); err != nil {
		return Gear{}, err
	}

	var item map[string]any
	for _, g := range raw {
		if s, _ := g["uuid"].(string); strings.EqualFold(s, uuid) {
			item = g
			break
		}
	}
	if item == nil {
		return Gear{}, fmt.Errorf("gear %q not found", uuid)
	}

	item["gearStatusName"] = status
	if status == "retired" {
		item["dateEnd"] = time.Now().In(time.Local).Format("2006-01-02") + "T00:00:00.0"
	} else {
		item["dateEnd"] = nil
	}

	var out gearRaw
	if err := c.PutJSON(ctx, "/gear-service/gear/"+url.PathEscape(uuid), nil, item, &out); err != nil {
		return Gear{}, err
	}
	return out.toGear(), nil
}

// FilterByStatus returns gear matching a status filter.
// status: "active" (default), "retired", or "" / "all" for everything.
func FilterByStatus(gears []Gear, status string) []Gear {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" || status == "all" {
		return gears
	}
	out := make([]Gear, 0, len(gears))
	for _, g := range gears {
		if strings.EqualFold(strings.TrimSpace(g.Status), status) {
			out = append(out, g)
		}
	}
	return out
}
