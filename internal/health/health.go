// Package health provides commands for health data retrieval from Garmin Connect.
// Supports: sleep, body-battery, stress, heart-rate, steps.
package health

// TODO: Implement API calls via client package
// TODO: Parse responses into typed structs
// TODO: Date range support

// SleepData represents sleep information for a given date.
type SleepData struct {
	Date           string  `json:"date"`
	DurationHours  float64 `json:"durationHours"`
	DeepSleepMin   int     `json:"deepSleepMin"`
	LightSleepMin  int     `json:"lightSleepMin"`
	RemSleepMin    int     `json:"remSleepMin"`
	AwakeMin       int     `json:"awakeMin"`
	SleepScore     int     `json:"sleepScore"`
}

// BodyBattery represents body battery data.
type BodyBattery struct {
	Date     string `json:"date"`
	Charged  int    `json:"charged"`
	Drained  int    `json:"drained"`
	High     int    `json:"high"`
	Low      int    `json:"low"`
}

// StressData represents stress level data.
type StressData struct {
	Date         string `json:"date"`
	OverallScore int    `json:"overallScore"`
	RestScore    int    `json:"restScore"`
	HighCount    int    `json:"highCount"`
	MediumCount  int    `json:"mediumCount"`
	LowCount     int    `json:"lowCount"`
}

// HeartRateData represents heart rate data.
type HeartRateData struct {
	Date          string `json:"date"`
	RestingHR     int    `json:"restingHR"`
	MaxHR         int    `json:"maxHR"`
	MinHR         int    `json:"minHR"`
}

// StepsData represents step count data.
type StepsData struct {
	Date       string `json:"date"`
	TotalSteps int    `json:"totalSteps"`
	Goal       int    `json:"goal"`
	Distance   int    `json:"distanceMeters"`
}
