package http

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// handleListServiceAccounts lists all service accounts with their API keys.
//
//	@Summary		List Service Accounts
//	@Tags			admin
//	@Produce		json
//	@Success		200		object	standardJsonResponse{data=[]thunderdome.ServiceAccount}
//	@Security		ApiKeyAuth
//	@Router			/admin/service-accounts [get]
func (s *Service) handleListServiceAccounts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		accounts, err := s.AdminDataSvc.ListServiceAccounts(ctx)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleListServiceAccounts error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		for _, sa := range accounts {
			keys, kErr := s.ApiKeyDataSvc.GetUserAPIKeys(ctx, sa.ID)
			if kErr == nil {
				sa.APIKeys = keys
			}
		}
		s.Success(w, r, http.StatusOK, accounts, nil)
	}
}

type serviceAccountCreateRequestBody struct {
	Name  string `json:"name" validate:"required,min=1,max=64"`
	Email string `json:"email" validate:"required,email"`
}

type serviceAccountCreateResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	APIKey string `json:"apiKey"` // plaintext — shown once
}

// handleCreateServiceAccount creates a service account and its first API key.
// The plaintext key is returned once and never again.
//
//	@Summary		Create Service Account
//	@Tags			admin
//	@Produce		json
//	@Param			account	body	serviceAccountCreateRequestBody	true	"service account"
//	@Success		200		object	standardJsonResponse{data=serviceAccountCreateResponse}
//	@Security		ApiKeyAuth
//	@Router			/admin/service-accounts [post]
func (s *Service) handleCreateServiceAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		var req serviceAccountCreateRequestBody
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

		sa, err := s.AdminDataSvc.CreateServiceAccount(ctx, req.Name, req.Email)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleCreateServiceAccount error", zap.Error(err),
				zap.String("session_user_id", sessionUserID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		apiKey, keyErr := s.ApiKeyDataSvc.GenerateAPIKey(ctx, sa.ID, "default")
		if keyErr != nil {
			s.Logger.Ctx(ctx).Error("handleCreateServiceAccount key error", zap.Error(keyErr),
				zap.String("service_account_id", sa.ID))
			s.Failure(w, r, http.StatusInternalServerError, keyErr)
			return
		}

		s.Success(w, r, http.StatusOK, serviceAccountCreateResponse{
			ID:     sa.ID,
			Name:   sa.Name,
			Email:  sa.Email,
			APIKey: apiKey.Key,
		}, nil)
	}
}

// handleGenerateServiceAccountKey mints an additional API key for a service
// account. Returns the plaintext key once.
//
//	@Summary		Generate Service Account API Key
//	@Tags			admin
//	@Produce		json
//	@Param			id	path	string	true	"service account id"
//	@Success		200		object	standardJsonResponse{data=thunderdome.APIKey}
//	@Security		ApiKeyAuth
//	@Router			/admin/service-accounts/{id}/apikeys [post]
func (s *Service) handleGenerateServiceAccountKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.PathValue("id")
		if err := validate.Var(id, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		isSA, err := s.AdminDataSvc.IsServiceAccount(ctx, id)
		if err != nil || !isSA {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "NOT_A_SERVICE_ACCOUNT"))
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		if req.Name == "" {
			req.Name = "key"
		}

		apiKey, keyErr := s.ApiKeyDataSvc.GenerateAPIKey(ctx, id, req.Name)
		if keyErr != nil {
			s.Logger.Ctx(ctx).Error("handleGenerateServiceAccountKey error", zap.Error(keyErr),
				zap.String("service_account_id", id))
			s.Failure(w, r, http.StatusInternalServerError, keyErr)
			return
		}
		s.Success(w, r, http.StatusOK, apiKey, nil)
	}
}

// handleDeleteServiceAccount deletes a service account (and cascades its keys,
// team memberships, etc).
//
//	@Summary		Delete Service Account
//	@Tags			admin
//	@Produce		json
//	@Param			id	path	string	true	"service account id"
//	@Success		200		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/admin/service-accounts/{id} [delete]
func (s *Service) handleDeleteServiceAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.PathValue("id")
		if err := validate.Var(id, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		if err := s.AdminDataSvc.DeleteServiceAccount(ctx, id); err != nil {
			s.Logger.Ctx(ctx).Error("handleDeleteServiceAccount error", zap.Error(err),
				zap.String("service_account_id", id))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}
		s.Success(w, r, http.StatusOK, nil, nil)
	}
}
