package training

import (
	"context"
	"fmt"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type StatusSummary struct {
	Date               string   `json:"date"`
	StatusPhrase       string   `json:"status_phrase,omitempty"`
	StatusID           *int     `json:"status_id,omitempty"`
	WeeklyTrainingLoad *float64 `json:"weekly_training_load,omitempty"`
	LoadLevelTrend     string   `json:"load_level_trend,omitempty"`
}

func GetStatus(ctx context.Context, c *client.Client, date string) (StatusSummary, error) {
	raw := map[string]any{}
	path := fmt.Sprintf("/mobile-gateway/usersummary/trainingstatus/latest/%s", date)
	if err := c.GetJSON(ctx, path, nil, &raw); err != nil {
		return StatusSummary{}, err
	}
	return summarizeStatus(date, raw), nil
}

func summarizeStatus(date string, raw map[string]any) StatusSummary {
	s := StatusSummary{Date: date}

	mr, _ := raw["mostRecentTrainingStatus"].(map[string]any)
	payload, _ := mr["payload"].(map[string]any)
	latest, _ := payload["latestTrainingStatusData"].(map[string]any)
	for _, v := range latest {
		if entry, ok := v.(map[string]any); ok {
			if phrase, ok := entry["trainingStatusFeedbackPhrase"].(string); ok {
				s.StatusPhrase = phrase
			}
			if id, ok := entry["trainingStatus"].(float64); ok {
				i := int(id)
				s.StatusID = &i
			}
			if wl, ok := entry["weeklyTrainingLoad"].(float64); ok {
				s.WeeklyTrainingLoad = &wl
			}
			if trend, ok := entry["loadLevelTrend"].(string); ok {
				s.LoadLevelTrend = trend
			}
			break
		}
	}

	if strings.TrimSpace(s.StatusPhrase) == "" && s.StatusID == nil {
		s.StatusPhrase = "—"
	}
	return s
}
