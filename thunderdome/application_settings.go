package thunderdome

import "time"

// ApplicationSettings holds workspace-wide toggles that an admin can change
// at runtime. Backed by a singleton row in thunderdome.application_settings.
type ApplicationSettings struct {
	RegistrationOpen bool      `json:"registration_open"`
	UpdatedBy        *string   `json:"updated_by,omitempty"`
	UpdatedDate      time.Time `json:"updated_date"`
}
