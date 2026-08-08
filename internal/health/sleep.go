package health

import (
	"context"
	"net/url"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type SleepDailyResponse struct {
	DailySleepDTO SleepDailyDTO `json:"dailySleepDTO"`
}

type SleepDailyDTO struct {
	CalendarDate             string      `json:"calendarDate"`
	SleepTimeSeconds         *int        `json:"sleepTimeSeconds"`
	DeepSleepSeconds         *int        `json:"deepSleepSeconds"`
	LightSleepSeconds        *int        `json:"lightSleepSeconds"`
	RemSleepSeconds          *int        `json:"remSleepSeconds"`
	AwakeSleepSeconds        *int        `json:"awakeSleepSeconds"`
	AverageSpO2Value         *float64    `json:"averageSpO2Value"`
	AverageRespirationValue  *float64    `json:"averageRespirationValue"`
	SleepStartTimestampGMT   *int64      `json:"sleepStartTimestampGMT"`
	SleepEndTimestampGMT     *int64      `json:"sleepEndTimestampGMT"`
	SleepStartTimestampLocal *int64      `json:"sleepStartTimestampLocal"`
	SleepEndTimestampLocal   *int64      `json:"sleepEndTimestampLocal"`
	SleepScores              SleepScores `json:"sleepScores"`
}

type SleepScores struct {
	Overall SleepScoreOverall `json:"overall"`
}

type SleepScoreOverall struct {
	Value *int `json:"value"`
}

type SleepSummary struct {
	Date              string   `json:"date"`
	Score             *int     `json:"score,omitempty"`
	TotalSleepSeconds *int     `json:"total_sleep_seconds,omitempty"`
	DeepSeconds       *int     `json:"deep_seconds,omitempty"`
	LightSeconds      *int     `json:"light_seconds,omitempty"`
	RemSeconds        *int     `json:"rem_seconds,omitempty"`
	AwakeSeconds      *int     `json:"awake_seconds,omitempty"`
	AvgSpO2           *float64 `json:"avg_spo2,omitempty"`
	AvgRespiration    *float64 `json:"avg_respiration,omitempty"`
	SleepStart        *string  `json:"sleep_start,omitempty"`
	SleepEnd          *string  `json:"sleep_end,omitempty"`
}

func garminLocalTimestamp(gmtMillis, localMillis *int64) *string {
	if gmtMillis == nil {
		return nil
	}

	offsetSeconds := 0
	if localMillis != nil {
		difference := (*localMillis - *gmtMillis) / 1000
		if difference >= -14*60*60 && difference <= 14*60*60 {
			offsetSeconds = int(difference)
		}
	}

	value := time.UnixMilli(*gmtMillis).
		In(time.FixedZone("Garmin local", offsetSeconds)).
		Format(time.RFC3339)
	return &value
}

func GetSleep(ctx context.Context, c *client.Client, date string) (SleepSummary, error) {
	var resp SleepDailyResponse
	q := url.Values{"date": {date}}
	if err := c.GetJSON(ctx, "/sleep-service/sleep/dailySleepData", q, &resp); err != nil {
		return SleepSummary{}, err
	}
	return resp.ToSummary(date), nil
}

func (r SleepDailyResponse) ToSummary(fallbackDate string) SleepSummary {
	date := r.DailySleepDTO.CalendarDate
	if date == "" {
		date = fallbackDate
	}
	return SleepSummary{
		Date:              date,
		Score:             r.DailySleepDTO.SleepScores.Overall.Value,
		TotalSleepSeconds: r.DailySleepDTO.SleepTimeSeconds,
		DeepSeconds:       r.DailySleepDTO.DeepSleepSeconds,
		LightSeconds:      r.DailySleepDTO.LightSleepSeconds,
		RemSeconds:        r.DailySleepDTO.RemSleepSeconds,
		AwakeSeconds:      r.DailySleepDTO.AwakeSleepSeconds,
		AvgSpO2:           r.DailySleepDTO.AverageSpO2Value,
		AvgRespiration:    r.DailySleepDTO.AverageRespirationValue,
		SleepStart:        garminLocalTimestamp(r.DailySleepDTO.SleepStartTimestampGMT, r.DailySleepDTO.SleepStartTimestampLocal),
		SleepEnd:          garminLocalTimestamp(r.DailySleepDTO.SleepEndTimestampGMT, r.DailySleepDTO.SleepEndTimestampLocal),
	}
}
