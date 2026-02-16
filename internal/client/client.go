// Package client provides an HTTP client for the Garmin Connect API.
package client

import (
	"net/http"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

// TODO: Implement API endpoints for health, activities, training
// TODO: Rate limiting and retry logic
// TODO: Response parsing and error handling

const (
	baseURL = "https://connect.garmin.com"
)

// Client is an authenticated HTTP client for Garmin Connect.
type Client struct {
	httpClient *http.Client
	session    *auth.Session
	baseURL    string
}

// New creates a new Garmin Connect API client.
func New(session *auth.Session) *Client {
	return &Client{
		httpClient: &http.Client{},
		session:    session,
		baseURL:    baseURL,
	}
}

// Get performs an authenticated GET request to the Garmin Connect API.
func (c *Client) Get(path string) (*http.Response, error) {
	// TODO: add auth headers, handle errors
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Do(req)
}
