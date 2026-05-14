package http

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/StevenWeathers/thunderdome-planning-poker/internal/linear"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// handleGetUserLinearInstances gets a list of Linear instances associated to user
//
//	@Summary		Get User Linear Instances
//	@Description	get list of Linear instances associated to user
//	@Tags			linear
//	@Produce		json
//	@Param			userId	path	string	true	"the user ID to find linear instances for"
//	@Success		200		object	standardJsonResponse{data=[]thunderdome.LinearInstance}
//	@Failure		500		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances [get]
func (s *Service) handleGetUserLinearInstances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		userID := r.PathValue("userId")

		idErr := validate.Var(userID, "required,uuid")
		if idErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, idErr.Error()))
			return
		}

		instances, err := s.LinearDataSvc.FindInstancesByUserID(ctx, userID)
		if err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleGetUserLinearInstances error", zap.Error(err), zap.String("entity_user_id", userID),
				zap.String("session_user_id", sessionUserID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, instances, nil)
	}
}

type linearInstanceRequestBody struct {
	Label       string `json:"label" validate:"required,min=1,max=120"`
	AccessToken string `json:"access_token" validate:"required"`
}

// handleLinearInstanceCreate creates a new Linear Instance
//
//	@Summary		Create Linear Instance
//	@Description	Creates a Linear Instance associated to user. Validates the access token against Linear and stores the workspace URL key.
//	@Tags			linear
//	@Produce		json
//	@Param			userId	path	string														true	"the user ID to associate linear instance to"
//	@Param			linear	body	linearInstanceRequestBody									true	"new linear_instance object"
//	@Success		200		object	standardJsonResponse{data=thunderdome.LinearInstance}		"returns new linear instance"
//	@Failure		500		object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances [post]
func (s *Service) handleLinearInstanceCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		userID := r.PathValue("userId")

		idErr := validate.Var(userID, "required,uuid")
		if idErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, idErr.Error()))
			return
		}

		req, err := readLinearInstanceRequest(r)
		if err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		workspaceURLKey := ""
		client, clientErr := linear.New(linear.Config{AccessToken: req.AccessToken})
		if clientErr == nil {
			ws, vErr := client.Viewer(ctx)
			if vErr != nil {
				s.Logger.Ctx(ctx).Error(
					"handleLinearInstanceCreate viewer error", zap.Error(vErr),
					zap.String("entity_user_id", userID), zap.String("session_user_id", sessionUserID))
				s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, "invalid Linear access token: "+vErr.Error()))
				return
			}
			workspaceURLKey = ws.URLKey
		}

		instance, err := s.LinearDataSvc.CreateInstance(ctx, userID, req.Label, workspaceURLKey, req.AccessToken)
		if err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleLinearInstanceCreate error", zap.Error(err), zap.String("entity_user_id", userID),
				zap.String("session_user_id", sessionUserID), zap.Stack("stacktrace"))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, instance, nil)
	}
}

// handleLinearInstanceUpdate updates a Linear Instance
//
//	@Summary		Update Linear Instance
//	@Description	Updates a Linear Instance associated to user
//	@Tags			linear
//	@Produce		json
//	@Param			userId		path	string														true	"the user ID linear instance associated to"
//	@Param			instanceId	path	string														true	"the linear_instance ID to update"
//	@Param			linear		body	linearInstanceRequestBody									true	"updated linear_instance object"
//	@Success		200			object	standardJsonResponse{data=thunderdome.LinearInstance}		"returns updated linear instance"
//	@Failure		500			object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances/{instanceId} [put]
func (s *Service) handleLinearInstanceUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		userID := r.PathValue("userId")
		instanceID := r.PathValue("instanceId")

		if err := validate.Var(instanceID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}
		if err := validate.Var(userID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		req, err := readLinearInstanceRequest(r)
		if err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		workspaceURLKey := ""
		client, clientErr := linear.New(linear.Config{AccessToken: req.AccessToken})
		if clientErr == nil {
			if ws, vErr := client.Viewer(ctx); vErr == nil {
				workspaceURLKey = ws.URLKey
			}
		}

		instance, err := s.LinearDataSvc.UpdateInstance(ctx, instanceID, req.Label, workspaceURLKey, req.AccessToken)
		if err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleLinearInstanceUpdate error", zap.Error(err), zap.String("entity_user_id", userID),
				zap.String("session_user_id", sessionUserID), zap.String("linear_instance_id", instanceID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, instance, nil)
	}
}

// handleLinearInstanceDelete deletes a Linear Instance
//
//	@Summary		Delete Linear Instance
//	@Description	Deletes a Linear Instance associated to user
//	@Tags			linear
//	@Produce		json
//	@Param			userId		path	string	true	"the user ID linear instance associated to"
//	@Param			instanceId	path	string	true	"the linear_instance ID to delete"
//	@Success		200			object	standardJsonResponse{}
//	@Failure		500			object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances/{instanceId} [delete]
func (s *Service) handleLinearInstanceDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		instanceID := r.PathValue("instanceId")
		if err := validate.Var(instanceID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		userID := r.PathValue("userId")
		if err := validate.Var(userID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		if err := s.LinearDataSvc.DeleteInstance(ctx, instanceID); err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleLinearInstanceDelete error", zap.Error(err), zap.String("entity_user_id", userID),
				zap.String("session_user_id", sessionUserID), zap.String("linear_instance_id", instanceID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, nil, nil)
	}
}

// handleLinearTeamsList queries Linear for the workspace's teams.
//
//	@Summary		List Linear Teams
//	@Description	Returns the list of teams visible to the configured Linear instance
//	@Tags			linear
//	@Produce		json
//	@Param			userId		path	string	true	"the user ID associated to linear instance"
//	@Param			instanceId	path	string	true	"the linear_instance ID to query"
//	@Success		200			object	standardJsonResponse{}
//	@Failure		500			object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances/{instanceId}/teams [get]
func (s *Service) handleLinearTeamsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		userID := r.PathValue("userId")
		if err := validate.Var(userID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		instanceID := r.PathValue("instanceId")
		if err := validate.Var(instanceID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		instance, err := s.LinearDataSvc.GetInstanceByID(ctx, instanceID)
		if err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleLinearTeamsList instance error", zap.Error(err),
				zap.String("entity_user_id", userID), zap.String("session_user_id", sessionUserID),
				zap.String("linear_instance_id", instanceID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		client, err := newLinearClient(instance)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleLinearTeamsList client error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		teams, err := client.Teams(ctx)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleLinearTeamsList teams error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, teams, nil)
	}
}

type linearIssueSearchRequestBody struct {
	Query   string `json:"query"`
	TeamKey string `json:"teamKey"`
	First   int    `json:"first"`
}

// handleLinearIssueSearch queries Linear API for Issues
//
//	@Summary		Query Linear for Issues
//	@Description	Queries Linear Instance API for Issues filtered by team and/or title text
//	@Tags			linear
//	@Produce		json
//	@Param			userId		path	string							true	"the user ID associated to linear instance"
//	@Param			instanceId	path	string							true	"the linear_instance ID to query"
//	@Param			linear		body	linearIssueSearchRequestBody	true	"issue search request"
//	@Success		200			object	standardJsonResponse{}
//	@Failure		500			object	standardJsonResponse{}
//	@Security		ApiKeyAuth
//	@Router			/users/{userId}/linear-instances/{instanceId}/issue-search [post]
func (s *Service) handleLinearIssueSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionUserID := ctx.Value(contextKeyUserID).(string)

		userID := r.PathValue("userId")
		if err := validate.Var(userID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		instanceID := r.PathValue("instanceId")
		if err := validate.Var(instanceID, "required,uuid"); err != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
			return
		}

		var req linearIssueSearchRequestBody
		body, bodyErr := io.ReadAll(r.Body)
		if bodyErr != nil {
			s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, bodyErr.Error()))
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				s.Failure(w, r, http.StatusBadRequest, Errorf(EINVALID, err.Error()))
				return
			}
		}

		instance, err := s.LinearDataSvc.GetInstanceByID(ctx, instanceID)
		if err != nil {
			s.Logger.Ctx(ctx).Error(
				"handleLinearIssueSearch instance error", zap.Error(err),
				zap.String("entity_user_id", userID), zap.String("session_user_id", sessionUserID),
				zap.String("linear_instance_id", instanceID))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		client, err := newLinearClient(instance)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleLinearIssueSearch client error", zap.Error(err))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		result, err := client.SearchIssues(ctx, req.TeamKey, req.Query, req.First)
		if err != nil {
			s.Logger.Ctx(ctx).Error("handleLinearIssueSearch search error", zap.Error(err),
				zap.String("team_key", req.TeamKey), zap.String("query", req.Query))
			s.Failure(w, r, http.StatusInternalServerError, err)
			return
		}

		s.Success(w, r, http.StatusOK, result, nil)
	}
}

func readLinearInstanceRequest(r *http.Request) (linearInstanceRequestBody, error) {
	var req linearInstanceRequestBody
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return req, err
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, err
	}
	if err := validate.Struct(req); err != nil {
		return req, err
	}
	return req, nil
}

func newLinearClient(instance thunderdome.LinearInstance) (*linear.Client, error) {
	return linear.New(linear.Config{AccessToken: instance.AccessToken})
}
