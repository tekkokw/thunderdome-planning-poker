package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"

	linearDB "github.com/StevenWeathers/thunderdome-planning-poker/internal/db/linear"
	"github.com/StevenWeathers/thunderdome-planning-poker/internal/linear"
)

// handleGetTeamLinearLink returns the Linear link for the team, or 404 when
// no link is configured.
//
//	@Summary		Get Team Linear Link
//	@Tags			linear
//	@Produce		json
//	@Param			teamId	path	string	true	"team id"
//	@Success		200		object	standardJsonResponse{data=thunderdome.TeamLinearLink}
//	@Failure		404		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/teams/{teamId}/linear-link [get]
func (s *Service) handleGetTeamLinearLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		teamID := r.PathValue("teamId")
		if err := validate.Var(teamID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		link, err := s.LinearDataSvc.GetTeamLink(ctx, teamID)
		if errors.Is(err, linearDB.ErrTeamLinkNotFound) {
			s.Failure(w, r, http.StatusNotFound, Errorf(ENOTFOUND, "TEAM_LINEAR_LINK_NOT_FOUND"))
			return
		}
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleGetTeamLinearLink error", zap.Error(err), zap.String("team_id", teamID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, link, nil)
	}
}

type teamLinearLinkRequestBody struct {
	LinearInstanceID string `json:"linear_instance_id" validate:"required,uuid"`
	LinearTeamID     string `json:"linear_team_id" validate:"required"`
	LinearTeamKey    string `json:"linear_team_key" validate:"required"`
	LinearTeamName   string `json:"linear_team_name"`
}

// handleUpsertTeamLinearLink creates or updates the Linear link for a team.
//
//	@Summary		Upsert Team Linear Link
//	@Tags			linear
//	@Produce		json
//	@Param			teamId	path	string						true	"team id"
//	@Param			link	body	teamLinearLinkRequestBody	true	"link payload"
//	@Success		200		object	standardJsonResponse{data=thunderdome.TeamLinearLink}
//	@Security		ApiKeyAuth
//	@Router			/teams/{teamId}/linear-link [put]
func (s *Service) handleUpsertTeamLinearLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)
		teamID := r.PathValue("teamId")
		if err := validate.Var(teamID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		var req teamLinearLinkRequestBody
		body, bodyErr := io.ReadAll(r.Body)
		if bodyErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, bodyErr.Error()))
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		if err := validate.Struct(req); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		// Confirm the linked instance belongs to the calling user. This isn't
		// a security boundary (any team admin could in theory link a different
		// user's instance if they knew the UUID), but it catches the common
		// mistake of pasting a stale id.
		inst, err := s.LinearDataSvc.GetInstanceByID(ctx, req.LinearInstanceID)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleUpsertTeamLinearLink instance lookup error", zap.Error(err))
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "LINEAR_INSTANCE_NOT_FOUND"))
			return
		}
		if inst.UserID != sessionUserID {
			s.Failure(w, r, http.StatusForbidden, Errorf(EUNAUTHORIZED, "LINEAR_INSTANCE_NOT_OWNED_BY_CALLER"))
			return
		}

		link, err := s.LinearDataSvc.UpsertTeamLink(ctx, teamID, req.LinearInstanceID, req.LinearTeamID, req.LinearTeamKey, req.LinearTeamName)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleUpsertTeamLinearLink error", zap.Error(err), zap.String("team_id", teamID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, link, nil)
	}
}

// handleDeleteTeamLinearLink removes the Linear link for a team.
//
//	@Summary		Delete Team Linear Link
//	@Tags			linear
//	@Produce		json
//	@Param			teamId	path	string	true	"team id"
//	@Success		200		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/teams/{teamId}/linear-link [delete]
func (s *Service) handleDeleteTeamLinearLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		teamID := r.PathValue("teamId")
		if err := validate.Var(teamID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		if err := s.LinearDataSvc.DeleteTeamLink(ctx, teamID); err != nil {
			s.Logger.Ctx(ctx).Error("handleDeleteTeamLinearLink error", zap.Error(err), zap.String("team_id", teamID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, nil, nil)
	}
}

// handleTeamLinearActiveCycle returns the linked Linear team's active cycle and
// the caller's open issues in it. Returns 404 when the team has no link or no
// active cycle.
//
//	@Summary		Get Team's Active Linear Cycle
//	@Tags			linear
//	@Produce		json
//	@Param			teamId	path	string	true	"team id"
//	@Success		200		object	standardJsonResponse{}
//	@Failure		404		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/teams/{teamId}/linear-link/active-cycle [get]
func (s *Service) handleTeamLinearActiveCycle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		teamID := r.PathValue("teamId")
		if err := validate.Var(teamID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		_, instance, err := s.LinearDataSvc.GetTeamLinkInstance(ctx, teamID)
		if errors.Is(err, linearDB.ErrTeamLinkNotFound) {
			s.Failure(w, r, http.StatusNotFound, Errorf(ENOTFOUND, "TEAM_LINEAR_LINK_NOT_FOUND"))
			return
		}
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleTeamLinearActiveCycle lookup error", zap.Error(err), zap.String("team_id", teamID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		client, err := linear.New(linear.Config{AccessToken: instance.AccessToken})
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleTeamLinearActiveCycle client error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		// Re-fetch the link to get the linear_team_id (GetTeamLinkInstance gave
		// it to us but we threw it away above for clarity; one extra query is
		// trivially cheap on a single keyed row).
		link, err := s.LinearDataSvc.GetTeamLink(ctx, teamID)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleTeamLinearActiveCycle link lookup error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		result, err := client.ActiveCycle(ctx, link.LinearTeamID)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleTeamLinearActiveCycle cycle fetch error", zap.Error(err),
				zap.String("team_id", teamID), zap.String("linear_team_id", link.LinearTeamID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, result, nil)
	}
}
