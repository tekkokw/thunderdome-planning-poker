package thunderdome

import (
	"time"
)

// LinearInstance represents a user's Linear workspace integration.
type LinearInstance struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Label           string    `json:"label"`
	WorkspaceURLKey string    `json:"workspace_url_key"`
	AccessToken     string    `json:"access_token"`
	CreatedDate     time.Time `json:"created_date"`
	UpdatedDate     time.Time `json:"updated_date"`
}

// TeamLinearLink connects a Thunderdome team to a Linear team via one user's
// stored Linear instance. The AccessToken on the linked instance is what drives
// cycle queries for the entire Thunderdome team — typically configured by a
// team lead or service account.
type TeamLinearLink struct {
	TeamID           string    `json:"team_id"`
	LinearInstanceID string    `json:"linear_instance_id"`
	LinearTeamID     string    `json:"linear_team_id"`
	LinearTeamKey    string    `json:"linear_team_key"`
	LinearTeamName   string    `json:"linear_team_name"`
	CreatedDate      time.Time `json:"created_date"`
	UpdatedDate      time.Time `json:"updated_date"`
}
