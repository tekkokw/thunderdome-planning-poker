package admin

import (
	"context"
	"fmt"

	"github.com/StevenWeathers/thunderdome-planning-poker/internal/db"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// CreateServiceAccount creates a non-human user that authenticates only via
// API keys. type is REGISTERED (so existing entity/team membership works) but
// is_service_account = true and no auth_credential row is created, so password
// and OAuth login are impossible for this account.
func (d *Service) CreateServiceAccount(ctx context.Context, name string, email string) (*thunderdome.ServiceAccount, error) {
	sa := &thunderdome.ServiceAccount{}
	sanitizedEmail := db.SanitizeEmail(email)

	err := d.DB.QueryRowContext(ctx,
		`INSERT INTO thunderdome.users (name, email, type, verified, is_service_account)
			VALUES ($1, $2, 'REGISTERED', true, true)
			RETURNING id, name, email, created_date, updated_date;`,
		name, sanitizedEmail,
	).Scan(&sa.ID, &sa.Name, &sa.Email, &sa.CreatedDate, &sa.UpdatedDate)
	if err != nil {
		return nil, fmt.Errorf("create service account: %v", err)
	}
	return sa, nil
}

// ListServiceAccounts returns all service accounts with their API keys.
func (d *Service) ListServiceAccounts(ctx context.Context) ([]*thunderdome.ServiceAccount, error) {
	accounts := make([]*thunderdome.ServiceAccount, 0)

	rows, err := d.DB.QueryContext(ctx,
		`SELECT u.id, u.name, COALESCE(u.email, ''), u.created_date, u.updated_date
			FROM thunderdome.users u
			WHERE u.is_service_account = true
			ORDER BY u.created_date;`,
	)
	if err != nil {
		return accounts, fmt.Errorf("list service accounts: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		sa := &thunderdome.ServiceAccount{}
		if err := rows.Scan(&sa.ID, &sa.Name, &sa.Email, &sa.CreatedDate, &sa.UpdatedDate); err != nil {
			return accounts, fmt.Errorf("scan service account: %v", err)
		}
		accounts = append(accounts, sa)
	}
	return accounts, nil
}

// IsServiceAccount reports whether a user id belongs to a service account.
func (d *Service) IsServiceAccount(ctx context.Context, userID string) (bool, error) {
	var isSA bool
	err := d.DB.QueryRowContext(ctx,
		`SELECT is_service_account FROM thunderdome.users WHERE id = $1;`, userID,
	).Scan(&isSA)
	if err != nil {
		return false, fmt.Errorf("is service account: %v", err)
	}
	return isSA, nil
}

// DeleteServiceAccount removes a service account. Guarded so a regular user
// can't be deleted through this path.
func (d *Service) DeleteServiceAccount(ctx context.Context, id string) error {
	res, err := d.DB.ExecContext(ctx,
		`DELETE FROM thunderdome.users WHERE id = $1 AND is_service_account = true;`, id,
	)
	if err != nil {
		return fmt.Errorf("delete service account: %v", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete service account rows: %v", err)
	}
	if n != 1 {
		return fmt.Errorf("service account not found")
	}
	return nil
}
