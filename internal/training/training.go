// Package training provides commands for training metrics from Garmin Connect.
// Supports: status, readiness, vo2max, hrv.
package training

// TODO: Implement API calls via client package
// TODO: Historical data / trends

// Status represents training status (Productive, Peaking, Recovery, etc.).
type Status struct {
	Date           string  `json:"date"`
	TrainingStatus string  `json:"trainingStatus"`
	TrainingLoad   float64 `json:"trainingLoad"`
}

// Readiness represents training readiness score.
type Readiness struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
	Level string `json:"level"`
}

// VO2Max represents VO2 max estimates.
type VO2Max struct {
	Date    string  `json:"date"`
	Running float64 `json:"running"`
	Cycling float64 `json:"cycling"`
}

// HRV represents heart rate variability data.
type HRV struct {
	Date       string  `json:"date"`
	WeeklyAvg  int     `json:"weeklyAvg"`
	LastNight  int     `json:"lastNight"`
	Status     string  `json:"status"`
	Baseline   int     `json:"baseline"`
}
