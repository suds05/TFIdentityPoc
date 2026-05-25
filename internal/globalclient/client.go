// ////////////////////////////////////////////////////////////
//
// Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License")
//
// A Client class for the Global Tier service.
// This is used by the Storage Tier to call the Global Tier's Discover API.
package globalclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DiscoverResult is the subset of the global discover response used by storage tiers.
type DiscoverResult struct {
	TeamIDs []string `json:"teamIds"`
}

// Client calls the global tier HTTP API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a client for the global tier base URL (e.g. http://global:8080).
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Discover proxies the caller's bearer token to GET /v1/discover.
// Returns the HTTP status from global and a parsed body on 200.
func (c *Client) Discover(ctx context.Context, bearerToken string) (DiscoverResult, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/discover", nil)
	if err != nil {
		return DiscoverResult{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DiscoverResult{}, 0, fmt.Errorf("global discover request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return DiscoverResult{}, resp.StatusCode, nil
	}

	var result DiscoverResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DiscoverResult{}, resp.StatusCode, fmt.Errorf("decode discover response: %w", err)
	}
	return result, resp.StatusCode, nil
}
