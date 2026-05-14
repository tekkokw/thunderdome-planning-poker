package linear

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/StevenWeathers/thunderdome-planning-poker/internal/db"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// ErrTeamLinkNotFound is returned when a team has no Linear link.
var ErrTeamLinkNotFound = errors.New("team linear link not found")

// GetTeamLink returns the Linear link for a given team, or ErrTeamLinkNotFound.
func (s *Service) GetTeamLink(ctx context.Context, teamID string) (thunderdome.TeamLinearLink, error) {
	link := thunderdome.TeamLinearLink{}
	err := s.DB.QueryRowContext(ctx,
		`SELECT team_id, linear_instance_id, linear_team_id, linear_team_key, linear_team_name, created_date, updated_date
			FROM thunderdome.team_linear_link WHERE team_id = $1;`,
		teamID,
	).Scan(
		&link.TeamID, &link.LinearInstanceID, &link.LinearTeamID, &link.LinearTeamKey, &link.LinearTeamName,
		&link.CreatedDate, &link.UpdatedDate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return link, ErrTeamLinkNotFound
	}
	if err != nil {
		return link, fmt.Errorf("get team linear link error: %v", err)
	}
	return link, nil
}

// UpsertTeamLink creates or replaces the Linear link for a team.
func (s *Service) UpsertTeamLink(ctx context.Context, teamID, instanceID, linearTeamID, linearTeamKey, linearTeamName string) (thunderdome.TeamLinearLink, error) {
	link := thunderdome.TeamLinearLink{}
	err := s.DB.QueryRowContext(ctx,
		`INSERT INTO thunderdome.team_linear_link
			(team_id, linear_instance_id, linear_team_id, linear_team_key, linear_team_name)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (team_id) DO UPDATE
			SET linear_instance_id = EXCLUDED.linear_instance_id,
				linear_team_id = EXCLUDED.linear_team_id,
				linear_team_key = EXCLUDED.linear_team_key,
				linear_team_name = EXCLUDED.linear_team_name,
				updated_date = now()
			RETURNING team_id, linear_instance_id, linear_team_id, linear_team_key, linear_team_name, created_date, updated_date;`,
		teamID, instanceID, linearTeamID, linearTeamKey, linearTeamName,
	).Scan(
		&link.TeamID, &link.LinearInstanceID, &link.LinearTeamID, &link.LinearTeamKey, &link.LinearTeamName,
		&link.CreatedDate, &link.UpdatedDate,
	)
	if err != nil {
		return link, fmt.Errorf("upsert team linear link error: %v", err)
	}
	return link, nil
}

// DeleteTeamLink removes the Linear link for a team. Returns nil even when the
// team had no link, so callers don't need to special-case absence.
func (s *Service) DeleteTeamLink(ctx context.Context, teamID string) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM thunderdome.team_linear_link WHERE team_id = $1;`, teamID); err != nil {
		return fmt.Errorf("delete team linear link error: %v", err)
	}
	return nil
}

// GetTeamLinkInstance returns the linked instance (with decrypted access token)
// for the given team, in a single query. Returns ErrTeamLinkNotFound when the
// team has no link.
func (s *Service) GetTeamLinkInstance(ctx context.Context, teamID string) (thunderdome.TeamLinearLink, thunderdome.LinearInstance, error) {
	link := thunderdome.TeamLinearLink{}
	instance := thunderdome.LinearInstance{}
	err := s.DB.QueryRowContext(ctx,
		`SELECT
			tll.team_id, tll.linear_instance_id, tll.linear_team_id, tll.linear_team_key, tll.linear_team_name,
			tll.created_date, tll.updated_date,
			li.id, li.user_id, li.label, li.workspace_url_key, li.access_token, li.created_date, li.updated_date
		FROM thunderdome.team_linear_link tll
		JOIN thunderdome.linear_instance li ON li.id = tll.linear_instance_id
		WHERE tll.team_id = $1;`,
		teamID,
	).Scan(
		&link.TeamID, &link.LinearInstanceID, &link.LinearTeamID, &link.LinearTeamKey, &link.LinearTeamName,
		&link.CreatedDate, &link.UpdatedDate,
		&instance.ID, &instance.UserID, &instance.Label, &instance.WorkspaceURLKey, &instance.AccessToken,
		&instance.CreatedDate, &instance.UpdatedDate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return link, instance, ErrTeamLinkNotFound
	}
	if err != nil {
		return link, instance, fmt.Errorf("get team link with instance error: %v", err)
	}
	plain, err := db.Decrypt(instance.AccessToken, s.AESHashKey)
	if err != nil {
		return link, instance, fmt.Errorf("decrypt linked instance token: %v", err)
	}
	instance.AccessToken = plain
	return link, instance, nil
}
