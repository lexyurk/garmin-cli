package workouts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

// ExportFIT downloads a workout's FIT file and streams it to w.
func ExportFIT(ctx context.Context, c *client.Client, workoutID int64, w io.Writer) error {
	resp, err := c.DoRaw(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/workout-service/workout/FIT/%d", workoutID),
		nil,
		nil,
		"",
		"*/*",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return fmt.Errorf("%w: %s: %s", auth.ErrNotAuthenticated, resp.Status, strings.TrimSpace(string(b)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return fmt.Errorf("garmin connectapi error: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

// Unschedule removes a scheduled workout occurrence by its schedule id.
//
// The schedule id is the workoutScheduleId returned by Schedule (or shown in
// the calendar), not the workout id.
func Unschedule(ctx context.Context, c *client.Client, scheduleID int64) error {
	return c.Delete(ctx, fmt.Sprintf("/workout-service/schedule/%d", scheduleID), nil)
}
