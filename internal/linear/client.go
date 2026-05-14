// Package linear provides Linear (linear.app) GraphQL integration.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// New creates a new Linear client from a Personal API key.
func New(config Config) (*Client, error) {
	if config.AccessToken == "" {
		return nil, fmt.Errorf("linear: access token is required")
	}
	return &Client{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		accessToken: config.AccessToken,
	}, nil
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors"`
}

func (c *Client) execute(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("linear: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("linear: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("linear: http error: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("linear: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("linear: status %d: %s", resp.StatusCode, string(raw))
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(raw, &gqlResp); err != nil {
		return fmt.Errorf("linear: decode response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("linear: %s", gqlResp.Errors[0].Message)
	}
	if out != nil {
		if err := json.Unmarshal(gqlResp.Data, out); err != nil {
			return fmt.Errorf("linear: decode data: %w", err)
		}
	}
	return nil
}
