package activities

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

type ExportType string

const (
	ExportGPX      ExportType = "gpx"
	ExportTCX      ExportType = "tcx"
	ExportOriginal ExportType = "original" // usually a .fit file
)

func (t ExportType) pathSegment() (string, bool) {
	switch t {
	case ExportGPX:
		return "gpx", true
	case ExportTCX:
		return "tcx", true
	case ExportOriginal:
		return "original", true
	default:
		return "", false
	}
}

// Export downloads an activity file (gpx/tcx/original) and streams it to w.
func Export(ctx context.Context, c *client.Client, activityID int64, exportType ExportType, w io.Writer) error {
	seg, ok := exportType.pathSegment()
	if !ok {
		return fmt.Errorf("unsupported export type %q", exportType)
	}

	resp, err := c.DoRaw(ctx, http.MethodGet, fmt.Sprintf("/download-service/export/%s/activity/%d", seg, activityID), nil, nil, "", "*/*")
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
