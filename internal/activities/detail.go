package activities

type DetailSummary struct {
	ID              int64
	Name            string
	Type            string
	StartTimeLocal  string
	DistanceMeters  float64
	DurationSeconds float64
	Calories        int
	AvgHR           int
	MaxHR           int
	ElevationGain   float64
	VO2Max          float64
	TrainingLoad    float64
}

func SummarizeDetail(id int64, raw map[string]any) DetailSummary {
	s := DetailSummary{ID: id}
	s.Name, _ = raw["activityName"].(string)
	s.StartTimeLocal, _ = raw["startTimeLocal"].(string)

	if at, ok := raw["activityType"].(map[string]any); ok {
		if tk, ok := at["typeKey"].(string); ok {
			s.Type = tk
		}
	}

	s.DistanceMeters = floatFromAny(raw["distance"])
	s.DurationSeconds = floatFromAny(raw["duration"])
	s.Calories = intFromAny(raw["calories"])
	s.AvgHR = intFromAny(raw["averageHR"])
	s.MaxHR = intFromAny(raw["maxHR"])
	s.ElevationGain = floatFromAny(raw["elevationGain"])
	s.VO2Max = floatFromAny(raw["vO2MaxValue"])
	s.TrainingLoad = floatFromAny(raw["activityTrainingLoad"])
	return s
}

type SplitSummary struct {
	DistanceMeters  float64 `json:"distance_meters"`
	DurationSeconds float64 `json:"duration_seconds"`
	AverageHR       int     `json:"average_hr,omitempty"`
	MaxHR           int     `json:"max_hr,omitempty"`
}

func ExtractSplits(raw map[string]any) []SplitSummary {
	splitsAny, ok := raw["splitSummaries"].([]any)
	if !ok || len(splitsAny) == 0 {
		return nil
	}

	out := make([]SplitSummary, 0, len(splitsAny))
	for _, item := range splitsAny {
		m, _ := item.(map[string]any)
		out = append(out, SplitSummary{
			DistanceMeters:  floatFromAny(m["distance"]),
			DurationSeconds: floatFromAny(m["duration"]),
			AverageHR:       intFromAny(m["averageHR"]),
			MaxHR:           intFromAny(m["maxHR"]),
		})
	}
	return out
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

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}
