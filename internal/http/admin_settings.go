package http

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// handleGetApplicationSettings returns the workspace-wide admin settings.
//
//	@Summary		Get Application Settings
//	@Tags			admin
//	@Produce		json
//	@Success		200		object	standardJsonResponse{data=thunderdome.ApplicationSettings}
//	@Security		ApiKeyAuth
//	@Router			/admin/application-settings [get]
func (s *Service) handleGetApplicationSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		settings, err := s.AdminDataSvc.GetApplicationSettings(ctx)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleGetApplicationSettings error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, settings, nil)
	}
}

type applicationSettingsRequestBody struct {
	RegistrationOpen bool `json:"registration_open"`
}

// handleUpdateApplicationSettings updates the workspace-wide admin settings.
//
//	@Summary		Update Application Settings
//	@Tags			admin
//	@Produce		json
//	@Param			settings	body	applicationSettingsRequestBody	true	"settings payload"
//	@Success		200			object	standardJsonResponse{data=thunderdome.ApplicationSettings}
//	@Security		ApiKeyAuth
//	@Router			/admin/application-settings [put]
func (s *Service) handleUpdateApplicationSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		var req applicationSettingsRequestBody
		body, bodyErr := io.ReadAll(r.Body)
		if bodyErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, bodyErr.Error()))
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		settings, err := s.AdminDataSvc.UpdateApplicationSettings(ctx, req.RegistrationOpen, sessionUserID)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleUpdateApplicationSettings error", zap.Error(err),
				zap.String("session_user_id", sessionUserID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, settings, nil)
	}
}
