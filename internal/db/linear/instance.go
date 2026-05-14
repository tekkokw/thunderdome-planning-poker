package linear

import (
	"context"
	"fmt"

	"github.com/StevenWeathers/thunderdome-planning-poker/internal/db"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// FindInstancesByUserID returns all LinearInstances for a given user ID.
func (s *Service) FindInstancesByUserID(ctx context.Context, userID string) ([]thunderdome.LinearInstance, error) {
	instances := make([]thunderdome.LinearInstance, 0)

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, user_id, label, workspace_url_key, access_token, created_date, updated_date
			FROM thunderdome.linear_instance WHERE user_id = $1 ORDER BY created_date;`,
		userID,
	)
	if err != nil {
		return instances, fmt.Errorf("find linear instance by user id query error: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		instance := thunderdome.LinearInstance{}
		if err := rows.Scan(
			&instance.ID, &instance.UserID, &instance.Label, &instance.WorkspaceURLKey,
			&instance.AccessToken, &instance.CreatedDate, &instance.UpdatedDate,
		); err != nil {
			return instances, fmt.Errorf("find linear instance by user id row scan error: %v", err)
		}
		instance.AccessToken, err = db.Decrypt(instance.AccessToken, s.AESHashKey)
		if err != nil {
			return instances, fmt.Errorf("error decrypting linear_instance %s access_token:  %v", instance.ID, err)
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetInstanceByID returns a LinearInstance for a given instance ID.
func (s *Service) GetInstanceByID(ctx context.Context, instanceID string) (thunderdome.LinearInstance, error) {
	instance := thunderdome.LinearInstance{}

	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, label, workspace_url_key, access_token, created_date, updated_date
			FROM thunderdome.linear_instance WHERE id = $1;`,
		instanceID,
	).Scan(
		&instance.ID, &instance.UserID, &instance.Label, &instance.WorkspaceURLKey,
		&instance.AccessToken, &instance.CreatedDate, &instance.UpdatedDate,
	)
	if err != nil {
		return instance, fmt.Errorf("error encountered getting linear_instance %s:  %v", instanceID, err)
	}
	instance.AccessToken, err = db.Decrypt(instance.AccessToken, s.AESHashKey)
	if err != nil {
		return instance, fmt.Errorf("error decrypting linear_instance %s access_token:  %v", instanceID, err)
	}

	return instance, nil
}

// CreateInstance creates a new LinearInstance.
func (s *Service) CreateInstance(ctx context.Context, userID string, label string, workspaceURLKey string, accessToken string) (thunderdome.LinearInstance, error) {
	instance := thunderdome.LinearInstance{}
	secureToken, err := db.Encrypt(accessToken, s.AESHashKey)
	if err != nil {
		return instance, fmt.Errorf("error encountered creating linear_instance:  %v", err)
	}

	err = s.DB.QueryRowContext(ctx,
		`INSERT INTO thunderdome.linear_instance
			(user_id, label, workspace_url_key, access_token)
			VALUES ($1, $2, $3, $4)
			RETURNING id, user_id, label, workspace_url_key, access_token, created_date, updated_date;`,
		userID, label, workspaceURLKey, secureToken,
	).Scan(
		&instance.ID, &instance.UserID, &instance.Label, &instance.WorkspaceURLKey,
		&instance.AccessToken, &instance.CreatedDate, &instance.UpdatedDate,
	)
	if err != nil {
		return instance, fmt.Errorf("error encountered creating linear_instance:  %v", err)
	}

	return instance, nil
}

// UpdateInstance updates an existing LinearInstance.
func (s *Service) UpdateInstance(ctx context.Context, instanceID string, label string, workspaceURLKey string, accessToken string) (thunderdome.LinearInstance, error) {
	instance := thunderdome.LinearInstance{}
	at, err := db.Encrypt(accessToken, s.AESHashKey)
	if err != nil {
		return instance, fmt.Errorf("error encountered updating linear_instance:  %v", err)
	}

	err = s.DB.QueryRowContext(ctx,
		`UPDATE thunderdome.linear_instance
			SET label = $2, workspace_url_key = $3, access_token = $4, updated_date = now()
			WHERE id = $1
			RETURNING id, user_id, label, workspace_url_key, access_token, created_date, updated_date;`,
		instanceID, label, workspaceURLKey, at,
	).Scan(
		&instance.ID, &instance.UserID, &instance.Label, &instance.WorkspaceURLKey,
		&instance.AccessToken, &instance.CreatedDate, &instance.UpdatedDate,
	)
	if err != nil {
		return instance, fmt.Errorf("error encountered updating linear_instance:  %v", err)
	}

	return instance, nil
}

// DeleteInstance deletes an existing LinearInstance.
func (s *Service) DeleteInstance(ctx context.Context, instanceID string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM thunderdome.linear_instance WHERE id = $1;`, instanceID)
	if err != nil {
		return fmt.Errorf("delete linear instance query error: %v", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete linear instance rows error: %v", err)
	}
	if rows != 1 {
		return fmt.Errorf("delete linear instance expected to affect 1 row, affected %d", rows)
	}

	return nil
}
