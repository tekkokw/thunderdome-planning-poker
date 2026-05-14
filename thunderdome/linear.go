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
