package linear

import (
	"context"
	"fmt"
	"time"
)

// Cycle represents a Linear cycle (iteration) for a team.
type Cycle struct {
	ID          string     `json:"id"`
	Number      int        `json:"number"`
	Name        string     `json:"name,omitempty"`
	StartsAt    *time.Time `json:"startsAt,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Progress    float64    `json:"progress"`
}

// ActiveCycleResult bundles the active cycle for a team and the viewer's
// in-progress issues within that cycle. Cycle is nil when the team has no
// active cycle.
type ActiveCycleResult struct {
	Team   *Team    `json:"team"`
	Cycle  *Cycle   `json:"cycle"`
	Issues []*Issue `json:"issues"`
}

const activeCycleQuery = `query ActiveCycle($teamId: String!) {
  team(id: $teamId) {
    id
    key
    name
    activeCycle {
      id
      number
      name
      startsAt
      endsAt
      completedAt
      progress
    }
  }
}`

const cycleIssuesQuery = `query CycleIssues($cycleId: ID!) {
  issues(
    filter: {
      cycle: { id: { eq: $cycleId } }
      assignee: { isMe: { eq: true } }
    }
    first: 100
    orderBy: updatedAt
  ) {
    nodes {
      id
      identifier
      title
      url
      priority
      estimate
      state { id name type }
      team { id key name }
    }
  }
}`

type activeCycleData struct {
	Team *struct {
		ID          string `json:"id"`
		Key         string `json:"key"`
		Name        string `json:"name"`
		ActiveCycle *struct {
			ID          string     `json:"id"`
			Number      int        `json:"number"`
			Name        string     `json:"name"`
			StartsAt    *time.Time `json:"startsAt"`
			EndsAt      *time.Time `json:"endsAt"`
			CompletedAt *time.Time `json:"completedAt"`
			Progress    float64    `json:"progress"`
		} `json:"activeCycle"`
	} `json:"team"`
}

type cycleIssuesData struct {
	Issues struct {
		Nodes []struct {
			ID         string         `json:"id"`
			Identifier string         `json:"identifier"`
			Title      string         `json:"title"`
			URL        string         `json:"url"`
			Priority   int            `json:"priority"`
			Estimate   *float64       `json:"estimate"`
			State      *WorkflowState `json:"state"`
			Team       *Team          `json:"team"`
		} `json:"nodes"`
	} `json:"issues"`
}

// ActiveCycle fetches the team's active cycle and the viewer's open issues in
// it. Returns a result with Cycle == nil when no cycle is active.
func (c *Client) ActiveCycle(ctx context.Context, linearTeamID string) (*ActiveCycleResult, error) {
	if linearTeamID == "" {
		return nil, fmt.Errorf("linear: team id is required")
	}

	var tData activeCycleData
	if err := c.execute(ctx, activeCycleQuery, map[string]any{"teamId": linearTeamID}, &tData); err != nil {
		return nil, err
	}
	if tData.Team == nil {
		return nil, fmt.Errorf("linear: team %s not found", linearTeamID)
	}

	result := &ActiveCycleResult{
		Team: &Team{ID: tData.Team.ID, Key: tData.Team.Key, Name: tData.Team.Name},
	}

	if tData.Team.ActiveCycle == nil {
		return result, nil
	}

	result.Cycle = &Cycle{
		ID:          tData.Team.ActiveCycle.ID,
		Number:      tData.Team.ActiveCycle.Number,
		Name:        tData.Team.ActiveCycle.Name,
		StartsAt:    tData.Team.ActiveCycle.StartsAt,
		EndsAt:      tData.Team.ActiveCycle.EndsAt,
		CompletedAt: tData.Team.ActiveCycle.CompletedAt,
		Progress:    tData.Team.ActiveCycle.Progress,
	}

	var iData cycleIssuesData
	if err := c.execute(ctx, cycleIssuesQuery, map[string]any{"cycleId": result.Cycle.ID}, &iData); err != nil {
		return nil, err
	}

	result.Issues = make([]*Issue, 0, len(iData.Issues.Nodes))
	for _, n := range iData.Issues.Nodes {
		result.Issues = append(result.Issues, &Issue{
			ID:         n.ID,
			Identifier: n.Identifier,
			Title:      n.Title,
			URL:        n.URL,
			Priority:   n.Priority,
			Estimate:   n.Estimate,
			State:      n.State,
			Team:       n.Team,
		})
	}

	return result, nil
}
