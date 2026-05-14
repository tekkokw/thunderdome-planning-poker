package linear

import (
	"context"
	"strings"
)

const viewerQuery = `query Viewer {
  viewer {
    name
    organization {
      id
      name
      urlKey
    }
  }
}`

type viewerData struct {
	Viewer struct {
		Name         string `json:"name"`
		Organization struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			URLKey string `json:"urlKey"`
		} `json:"organization"`
	} `json:"viewer"`
}

// Viewer returns the authenticated user's workspace info; used to validate the
// access token at instance-creation time and to capture the workspace URL key.
func (c *Client) Viewer(ctx context.Context) (*Workspace, error) {
	var data viewerData
	if err := c.execute(ctx, viewerQuery, nil, &data); err != nil {
		return nil, err
	}
	return &Workspace{
		ID:       data.Viewer.Organization.ID,
		Name:     data.Viewer.Organization.Name,
		URLKey:   data.Viewer.Organization.URLKey,
		UserName: data.Viewer.Name,
	}, nil
}

const teamsQuery = `query Teams {
  teams(first: 250) {
    nodes {
      id
      key
      name
    }
  }
}`

type teamsData struct {
	Teams struct {
		Nodes []*Team `json:"nodes"`
	} `json:"teams"`
}

// Teams returns the list of teams visible to the authenticated user.
func (c *Client) Teams(ctx context.Context) (*TeamsResult, error) {
	var data teamsData
	if err := c.execute(ctx, teamsQuery, nil, &data); err != nil {
		return nil, err
	}
	return &TeamsResult{Teams: data.Teams.Nodes}, nil
}

const issuesQuery = `query Issues($filter: IssueFilter, $first: Int) {
  issues(filter: $filter, first: $first, orderBy: updatedAt) {
    nodes {
      id
      identifier
      title
      description
      url
      priority
      estimate
      state { id name type }
      team { id key name }
      labels(first: 25) { nodes { id name } }
    }
  }
}`

type issuesData struct {
	Issues struct {
		Nodes []struct {
			ID          string         `json:"id"`
			Identifier  string         `json:"identifier"`
			Title       string         `json:"title"`
			Description string         `json:"description"`
			URL         string         `json:"url"`
			Priority    int            `json:"priority"`
			Estimate    *float64       `json:"estimate"`
			State       *WorkflowState `json:"state"`
			Team        *Team          `json:"team"`
			Labels      struct {
				Nodes []IssueLabel `json:"nodes"`
			} `json:"labels"`
		} `json:"nodes"`
	} `json:"issues"`
}

// SearchIssues queries Linear for issues matching the given filters. teamKey
// scopes results to a single team; query does a fuzzy match on title; if both
// are empty, returns the most recently updated issues across the workspace.
func (c *Client) SearchIssues(ctx context.Context, teamKey string, query string, first int) (*IssueSearchResult, error) {
	if first <= 0 || first > 250 {
		first = 50
	}

	filter := map[string]any{}
	teamKey = strings.TrimSpace(teamKey)
	if teamKey != "" {
		filter["team"] = map[string]any{
			"key": map[string]any{"eq": teamKey},
		}
	}
	query = strings.TrimSpace(query)
	if query != "" {
		filter["title"] = map[string]any{"containsIgnoreCase": query}
	}

	vars := map[string]any{"first": first}
	if len(filter) > 0 {
		vars["filter"] = filter
	}

	var data issuesData
	if err := c.execute(ctx, issuesQuery, vars, &data); err != nil {
		return nil, err
	}

	issues := make([]*Issue, 0, len(data.Issues.Nodes))
	for _, n := range data.Issues.Nodes {
		issues = append(issues, &Issue{
			ID:          n.ID,
			Identifier:  n.Identifier,
			Title:       n.Title,
			Description: n.Description,
			URL:         n.URL,
			Priority:    n.Priority,
			Estimate:    n.Estimate,
			State:       n.State,
			Team:        n.Team,
			Labels:      n.Labels.Nodes,
		})
	}
	return &IssueSearchResult{Total: len(issues), Issues: issues}, nil
}
