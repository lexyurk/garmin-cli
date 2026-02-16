// Package activities provides commands for activity management in Garmin Connect.
// Supports: list, get, splits.
package activities

// TODO: Implement API calls via client package
// TODO: Activity type filtering
// TODO: Date range filtering

// Activity represents a Garmin Connect activity.
type Activity struct {
	ID           int64   `json:"activityId"`
	Name         string  `json:"activityName"`
	Type         string  `json:"activityType"`
	StartTime    string  `json:"startTimeLocal"`
	Duration     float64 `json:"duration"`
	Distance     float64 `json:"distance"`
	Calories     int     `json:"calories"`
	AvgHR        int     `json:"averageHR"`
	MaxHR        int     `json:"maxHR"`
	ElevGain     float64 `json:"elevationGain"`
}

// Split represents a lap or split within an activity.
type Split struct {
	Number    int     `json:"splitNumber"`
	Distance  float64 `json:"distance"`
	Duration  float64 `json:"duration"`
	AvgPace   string  `json:"averagePace"`
	AvgHR     int     `json:"averageHR"`
	ElevGain  float64 `json:"elevationGain"`
}

// ListOptions holds parameters for listing activities.
type ListOptions struct {
	Limit  int
	After  string
	Before string
	Type   string
}
