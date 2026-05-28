package training

import (
	"context"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type FitnessAgeSummary struct {
	Date             string             `json:"date"`
	FitnessAge       *float64           `json:"fitness_age,omitempty"`
	ChronologicalAge *float64           `json:"chronological_age,omitempty"`
	AchievableAge    *float64           `json:"achievable_fitness_age,omitempty"`
	PreviousAge      *float64           `json:"previous_fitness_age,omitempty"`
	Components       map[string]float64 `json:"components,omitempty"`
}

type fitnessAgeRaw struct {
	FitnessAge           *float64 `json:"fitnessAge"`
	ChronologicalAge     *float64 `json:"chronologicalAge"`
	AchievableFitnessAge *float64 `json:"achievableFitnessAge"`
	PreviousFitnessAge   *float64 `json:"previousFitnessAge"`
	Components           map[string]struct {
		Value *float64 `json:"value"`
	} `json:"components"`
}

func GetFitnessAge(ctx context.Context, c *client.Client, date string) (FitnessAgeSummary, error) {
	var raw fitnessAgeRaw
	if err := c.GetJSON(ctx, fmt.Sprintf("/fitnessage-service/fitnessage/%s", date), nil, &raw); err != nil {
		return FitnessAgeSummary{}, err
	}
	return summarizeFitnessAge(date, raw), nil
}

func summarizeFitnessAge(date string, raw fitnessAgeRaw) FitnessAgeSummary {
	s := FitnessAgeSummary{
		Date:             date,
		FitnessAge:       raw.FitnessAge,
		ChronologicalAge: raw.ChronologicalAge,
		AchievableAge:    raw.AchievableFitnessAge,
		PreviousAge:      raw.PreviousFitnessAge,
	}
	if len(raw.Components) > 0 {
		s.Components = make(map[string]float64, len(raw.Components))
		for k, v := range raw.Components {
			if v.Value != nil {
				s.Components[k] = *v.Value
			}
		}
	}
	return s
}
