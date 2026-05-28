// Package devices reads the user's registered Garmin devices.
package devices

import (
	"context"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type Device struct {
	DeviceID     int64  `json:"device_id,omitempty"`
	Name         string `json:"name,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	PartNumber   string `json:"part_number,omitempty"`
}

type deviceRaw struct {
	DeviceID           int64  `json:"deviceId"`
	ProductDisplayName string `json:"productDisplayName"`
	DisplayName        string `json:"displayName"`
	SerialNumber       string `json:"serialNumber"`
	PartNumber         string `json:"partNumber"`
}

func (d deviceRaw) toDevice() Device {
	name := strings.TrimSpace(d.ProductDisplayName)
	if name == "" {
		name = strings.TrimSpace(d.DisplayName)
	}
	return Device{
		DeviceID:     d.DeviceID,
		Name:         name,
		SerialNumber: d.SerialNumber,
		PartNumber:   d.PartNumber,
	}
}

const devicesPath = "/device-service/deviceregistration/devices"

// List returns the user's registered devices.
func List(ctx context.Context, c *client.Client) ([]Device, error) {
	var raw []deviceRaw
	if err := c.GetJSON(ctx, devicesPath, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(raw))
	for _, d := range raw {
		out = append(out, d.toDevice())
	}
	return out, nil
}

// ListRaw returns the raw device payloads (for full-fidelity JSON output).
func ListRaw(ctx context.Context, c *client.Client) ([]map[string]any, error) {
	var raw []map[string]any
	if err := c.GetJSON(ctx, devicesPath, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
