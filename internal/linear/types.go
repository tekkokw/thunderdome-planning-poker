package linear

import "net/http"

// Endpoint is the Linear GraphQL API endpoint.
const Endpoint = "https://api.linear.app/graphql"

// Config is the configuration for a Linear client.
type Config struct {
	AccessToken string `json:"access_token"`
}

// Client wraps an authenticated HTTP client for talking to Linear's GraphQL API.
type Client struct {
	httpClient  *http.Client
	accessToken string
}

// Team represents a Linear team.
type Team struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// WorkflowState represents an issue workflow state (Todo, In Progress, Done, etc).
type WorkflowState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// IssueLabel represents a Linear label on an issue.
type IssueLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Issue represents a Linear issue.
type Issue struct {
	ID          string         `json:"id"`
	Identifier  string         `json:"identifier"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	Priority    int            `json:"priority"`
	Estimate    *float64       `json:"estimate"`
	State       *WorkflowState `json:"state"`
	Team        *Team          `json:"team"`
	Labels      []IssueLabel   `json:"labels"`
}

// Workspace represents the authenticated Linear workspace.
type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URLKey   string `json:"url_key"`
	UserName string `json:"user_name"`
}

// IssueSearchResult is the result of an issue search.
type IssueSearchResult struct {
	Total  int      `json:"total"`
	Issues []*Issue `json:"issues"`
}

// TeamsResult is the result of a teams query.
type TeamsResult struct {
	Teams []*Team `json:"teams"`
}
