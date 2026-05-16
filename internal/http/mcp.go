package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/StevenWeathers/thunderdome-planning-poker/internal/linear"
	"github.com/StevenWeathers/thunderdome-planning-poker/thunderdome"
)

// Minimal in-tree Model Context Protocol server.
//
// Transport: Streamable HTTP (single POST endpoint, JSON-RPC 2.0 request →
// JSON response). We don't open server→client SSE streams, so GET is 405.
// Auth: reuses the standard userOnly middleware, so an agent authenticates
// with the same X-API-Key it would use for the REST API (typically a service
// account's key). Every tool is scoped to that authenticated principal.

const (
	mcpProtocolVersion = "2025-03-26"
	mcpServerName      = "thunderdome"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type mcpToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpToolContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	handler     func(ctx context.Context, userID string, args map[string]any) (any, error)
}

func obj(props map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func strProp(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

// mcpRequireTeamMember returns nil only if userID is a member of teamID.
func (s *Service) mcpRequireTeamMember(ctx context.Context, userID, teamID string) error {
	if err := validate.Var(teamID, "required,uuid"); err != nil {
		return fmt.Errorf("invalid team_id")
	}
	if _, err := s.TeamDataSvc.TeamUserRoleByUserID(ctx, userID, teamID); err != nil {
		return fmt.Errorf("not a member of team %s", teamID)
	}
	return nil
}

func (s *Service) buildMCPTools() map[string]mcpTool {
	tools := map[string]mcpTool{}
	add := func(t mcpTool) { tools[t.Name] = t }

	add(mcpTool{
		Name:        "whoami",
		Description: "Return the authenticated principal (the API key's user/service account).",
		InputSchema: obj(map[string]any{}),
		handler: func(ctx context.Context, userID string, _ map[string]any) (any, error) {
			u, err := s.UserDataSvc.GetUserByID(ctx, userID)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"id":                 u.ID,
				"name":               u.Name,
				"email":              u.Email,
				"type":               u.Type,
				"is_service_account": u.IsServiceAccount,
			}, nil
		},
	})

	add(mcpTool{
		Name:        "list_teams",
		Description: "List the teams the authenticated principal belongs to.",
		InputSchema: obj(map[string]any{}),
		handler: func(ctx context.Context, userID string, _ map[string]any) (any, error) {
			teams := s.TeamDataSvc.TeamListByUser(ctx, userID, 1000, 0)
			out := make([]map[string]any, 0, len(teams))
			for _, t := range teams {
				out = append(out, map[string]any{
					"id":   t.ID,
					"name": t.Name,
					"role": t.Role,
				})
			}
			return map[string]any{"teams": out}, nil
		},
	})

	add(mcpTool{
		Name:        "list_team_checkins",
		Description: "List a team's daily checkins for a date (default: today UTC). Requires team membership.",
		InputSchema: obj(map[string]any{
			"team_id": strProp("UUID of the team"),
			"date":    strProp("Date as YYYY-MM-DD; defaults to today (UTC)"),
		}, "team_id"),
		handler: func(ctx context.Context, userID string, args map[string]any) (any, error) {
			teamID, _ := args["team_id"].(string)
			if err := s.mcpRequireTeamMember(ctx, userID, teamID); err != nil {
				return nil, err
			}
			date, _ := args["date"].(string)
			if date == "" {
				date = time.Now().UTC().Format("2006-01-02")
			}
			checkins, err := s.CheckinDataSvc.CheckinList(ctx, teamID, date)
			if err != nil {
				return nil, err
			}
			return map[string]any{"date": date, "checkins": checkins}, nil
		},
	})

	add(mcpTool{
		Name:        "get_team_active_cycle",
		Description: "Get the linked Linear team's active cycle and the configured instance owner's open issues. Requires team membership and a configured Linear link.",
		InputSchema: obj(map[string]any{
			"team_id": strProp("UUID of the team"),
		}, "team_id"),
		handler: func(ctx context.Context, userID string, args map[string]any) (any, error) {
			teamID, _ := args["team_id"].(string)
			if err := s.mcpRequireTeamMember(ctx, userID, teamID); err != nil {
				return nil, err
			}
			link, instance, err := s.LinearDataSvc.GetTeamLinkInstance(ctx, teamID)
			if err != nil {
				return nil, fmt.Errorf("no Linear link configured for this team")
			}
			client, err := linear.New(linear.Config{AccessToken: instance.AccessToken})
			if err != nil {
				return nil, err
			}
			result, err := client.ActiveCycle(ctx, link.LinearTeamID)
			if err != nil {
				return nil, err
			}
			return result, nil
		},
	})

	add(mcpTool{
		Name: "post_team_checkin",
		Description: "Create a daily checkin for a team. Defaults the subject to the authenticated principal; " +
			"posting for another user requires the principal to be an admin or service account.",
		InputSchema: obj(map[string]any{
			"team_id":   strProp("UUID of the team"),
			"user_id":   strProp("Subject user UUID; defaults to the authenticated principal"),
			"yesterday": strProp("What was done yesterday (markdown)"),
			"today":     strProp("What will be done today (markdown)"),
			"blockers":  strProp("Current blockers (markdown); empty if none"),
			"discuss":   strProp("Discussion items (markdown)"),
			"goals_met": boolProp("Whether prior goals were met (default true)"),
		}, "team_id"),
		handler: func(ctx context.Context, userID string, args map[string]any) (any, error) {
			teamID, _ := args["team_id"].(string)
			if err := validate.Var(teamID, "required,uuid"); err != nil {
				return nil, fmt.Errorf("invalid team_id")
			}

			subject, _ := args["user_id"].(string)
			if subject == "" {
				subject = userID
			}
			postedBy := ""
			if subject != userID {
				isSA, _ := s.AdminDataSvc.IsServiceAccount(ctx, userID)
				u, uErr := s.UserDataSvc.GetUserByID(ctx, userID)
				isAdmin := uErr == nil && u.Type == thunderdome.AdminUserType
				if !isSA && !isAdmin {
					return nil, fmt.Errorf("not permitted to post a checkin for another user")
				}
				postedBy = userID
			}

			goalsMet := true
			if v, ok := args["goals_met"].(bool); ok {
				goalsMet = v
			}
			str := func(k string) string { v, _ := args[k].(string); return v }

			err := s.CheckinDataSvc.CheckinCreate(
				ctx, teamID, subject, "",
				str("yesterday"), str("today"), str("blockers"), str("discuss"),
				goalsMet, "", postedBy,
			)
			if err != nil {
				return nil, err
			}
			return map[string]any{"status": "created", "team_id": teamID, "user_id": subject}, nil
		},
	})

	return tools
}

func writeJSONRPC(w http.ResponseWriter, resp jsonRPCResponse) {
	resp.JSONRPC = "2.0"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func rpcErr(id json.RawMessage, code int, msg string) jsonRPCResponse {
	return jsonRPCResponse{ID: id, Error: &jsonRPCError{Code: code, Message: msg}}
}

// handleMCP serves the MCP JSON-RPC endpoint. Mounted behind userOnly so the
// authenticated user id is in context.
//
//	@Summary		MCP endpoint
//	@Description	Model Context Protocol JSON-RPC 2.0 endpoint (Streamable HTTP transport)
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Router			/mcp [post]
func (s *Service) handleMCP() http.HandlerFunc {
	tools := s.buildMCPTools()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := ctx.Value(contextKeyUserID).(string)

		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONRPC(w, rpcErr(nil, -32700, "parse error"))
			return
		}

		// Notifications (no id) get acknowledged with 202 and no body.
		isNotification := len(req.ID) == 0
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		switch req.Method {
		case "initialize":
			writeJSONRPC(w, jsonRPCResponse{ID: req.ID, Result: map[string]any{
				"protocolVersion": mcpProtocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{
					"name":    mcpServerName,
					"version": s.UIConfig.AppConfig.AppVersion,
				},
			}})

		case "ping":
			writeJSONRPC(w, jsonRPCResponse{ID: req.ID, Result: map[string]any{}})

		case "tools/list":
			list := make([]map[string]any, 0, len(tools))
			for _, t := range tools {
				list = append(list, map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"inputSchema": t.InputSchema,
				})
			}
			writeJSONRPC(w, jsonRPCResponse{ID: req.ID, Result: map[string]any{"tools": list}})

		case "tools/call":
			var call struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &call); err != nil {
				writeJSONRPC(w, rpcErr(req.ID, -32602, "invalid params"))
				return
			}
			tool, ok := tools[call.Name]
			if !ok {
				writeJSONRPC(w, rpcErr(req.ID, -32602, "unknown tool: "+call.Name))
				return
			}
			if call.Arguments == nil {
				call.Arguments = map[string]any{}
			}
			data, err := tool.handler(ctx, userID, call.Arguments)
			if err != nil {
				// Tool errors are reported in-band per MCP spec.
				s.Logger.Ctx(ctx).Warn("mcp tool error",
					zap.String("tool", call.Name), zap.String("user_id", userID), zap.Error(err))
				writeJSONRPC(w, jsonRPCResponse{ID: req.ID, Result: mcpToolResult{
					Content: []mcpToolContent{{Type: "text", Text: err.Error()}},
					IsError: true,
				}})
				return
			}
			payload, _ := json.MarshalIndent(data, "", "  ")
			writeJSONRPC(w, jsonRPCResponse{ID: req.ID, Result: mcpToolResult{
				Content: []mcpToolContent{{Type: "text", Text: string(payload)}},
			}})

		default:
			writeJSONRPC(w, rpcErr(req.ID, -32601, "method not found: "+req.Method))
		}
	}
}
