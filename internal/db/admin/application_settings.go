package admin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// GetApplicationSettings returns the singleton application_settings row.
func (d *Service) GetApplicationSettings(ctx context.Context) (thunderdome.ApplicationSettings, error) {
	s := thunderdome.ApplicationSettings{}
	var updatedBy sql.NullString
	err := d.DB.QueryRowContext(ctx,
		`SELECT registration_open, updated_by, updated_date
			FROM thunderdome.application_settings WHERE id = 1;`,
	).Scan(&s.RegistrationOpen, &updatedBy, &s.UpdatedDate)
	if err != nil {
		return s, fmt.Errorf("get application settings: %v", err)
	}
	if updatedBy.Valid {
		v := updatedBy.String
		s.UpdatedBy = &v
	}
	return s, nil
}

// UpdateApplicationSettings overwrites the singleton settings row.
func (d *Service) UpdateApplicationSettings(ctx context.Context, registrationOpen bool, actorUserID string) (thunderdome.ApplicationSettings, error) {
	s := thunderdome.ApplicationSettings{}
	var updatedBy sql.NullString
	err := d.DB.QueryRowContext(ctx,
		`UPDATE thunderdome.application_settings
			SET registration_open = $1, updated_by = $2, updated_date = now()
			WHERE id = 1
			RETURNING registration_open, updated_by, updated_date;`,
		registrationOpen, actorUserID,
	).Scan(&s.RegistrationOpen, &updatedBy, &s.UpdatedDate)
	if err != nil {
		return s, fmt.Errorf("update application settings: %v", err)
	}
	if updatedBy.Valid {
		v := updatedBy.String
		s.UpdatedBy = &v
	}
	return s, nil
}

// IsRegistrationOpen is a small helper for the register handler hot path.
func (d *Service) IsRegistrationOpen(ctx context.Context) (bool, error) {
	var open bool
	err := d.DB.QueryRowContext(ctx,
		`SELECT registration_open FROM thunderdome.application_settings WHERE id = 1;`,
	).Scan(&open)
	if err != nil {
		return false, fmt.Errorf("read registration_open: %v", err)
	}
	return open, nil
}

// CountActiveAccounts returns the number of users that count toward the
// "first user becomes admin" bootstrap (registered + admin types).
func (d *Service) CountActiveAccounts(ctx context.Context) (int, error) {
	var n int
	err := d.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM thunderdome.users WHERE type IN ('REGISTERED', 'ADMIN');`,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active accounts: %v", err)
	}
	return n, nil
}
