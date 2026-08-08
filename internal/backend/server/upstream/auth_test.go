package upstream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cursor/gen/aiserverv1"
	"cursor/internal/backend/server"

	"google.golang.org/protobuf/proto"
)

func TestMockDevSessionTokenActionSupportsCursorDevLoginModes(t *testing.T) {
	testCases := []struct {
		name  string
		query string
		plan  string
		trial bool
	}{
		{name: "default", query: "", plan: "ultra"},
		{name: "free", query: "?plan=free", plan: "free"},
		{name: "pro trial", query: "?plan=pro&trial=true", plan: "pro", trial: true},
		{name: "pro", query: "?plan=pro", plan: "pro"},
		{name: "pro plus trial", query: "?plan=pro_plus&trial=true", plan: "pro_plus", trial: true},
		{name: "pro plus", query: "?plan=pro_plus", plan: "pro_plus"},
		{name: "ultra", query: "?plan=ultra", plan: "ultra"},
		{name: "enterprise", query: "?plan=enterprise", plan: "enterprise"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://local/auth/cursor_dev_session_token"+testCase.query, nil)
			recorder := httptest.NewRecorder()
			handler := MockDevSessionTokenAction(Dependencies{}, CompatRouteConfig{Name: "dev_login", StatusCode: http.StatusOK})
			if err := handler(&server.Context{Writer: recorder, Request: request}); err != nil {
				t.Fatalf("dev login handler: %v", err)
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}

			var response struct {
				AccessToken  string `json:"accessToken"`
				RefreshToken string `json:"refreshToken"`
				AuthID       string `json:"authId"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.AccessToken == "" || response.RefreshToken != response.AccessToken {
				t.Fatalf("unexpected tokens: access=%q refresh=%q", response.AccessToken, response.RefreshToken)
			}
			claims, ok := localDevClaimsFromAuthorization("Bearer " + response.AccessToken)
			if !ok {
				t.Fatal("response access token is not a local dev JWT")
			}
			if claims.Plan != testCase.plan || claims.Trial != testCase.trial {
				t.Fatalf("claims: got plan=%q trial=%v, want plan=%q trial=%v", claims.Plan, claims.Trial, testCase.plan, testCase.trial)
			}
			if response.AuthID != claims.Subject || claims.ExpiresAt <= time.Now().Unix() {
				t.Fatalf("unexpected identity claims: response=%+v claims=%+v", response, claims)
			}
		})
	}
}

func TestMockDevSessionTokenActionUsesRequestedEmail(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://local/auth/cursor_dev_session_token?plan=pro&email=dev%2Bcursor%40example.com", nil)
	recorder := httptest.NewRecorder()
	handler := MockDevSessionTokenAction(Dependencies{}, CompatRouteConfig{Name: "dev_login", StatusCode: http.StatusOK})
	if err := handler(&server.Context{Writer: recorder, Request: request}); err != nil {
		t.Fatalf("dev login handler: %v", err)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	claims, ok := localDevClaimsFromAuthorization(response["accessToken"])
	if !ok || claims.Email != "dev+cursor@example.com" {
		t.Fatalf("unexpected email claims: %+v", claims)
	}
}

func TestMockDevSessionTokenActionRejectsUnsupportedOptions(t *testing.T) {
	for _, query := range []string{"?plan=business", "?plan=ultra&trial=true", "?plan=pro&trial=maybe"} {
		request := httptest.NewRequest(http.MethodGet, "http://local/auth/cursor_dev_session_token"+query, nil)
		recorder := httptest.NewRecorder()
		handler := MockDevSessionTokenAction(Dependencies{}, CompatRouteConfig{Name: "dev_login", StatusCode: http.StatusOK})
		if err := handler(&server.Context{Writer: recorder, Request: request}); err != nil {
			t.Fatalf("dev login handler for %q: %v", query, err)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status for %q: got %d, want %d", query, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestEnterpriseDevSessionProvidesBillableTeam(t *testing.T) {
	token, _, err := buildLocalDevSessionToken("enterprise", false, "enterprise@example.com", time.Now())
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	reqCtx := authRequestContext(http.MethodPost, "/aiserver.v1.DashboardService/GetTeams", "", token)
	payload, err := buildDashboardTeamsPayload(reqCtx)
	if err != nil {
		t.Fatalf("build teams: %v", err)
	}
	encoded, err := encodeMockProto("aiserver.v1.GetTeamsResponse", payload)
	if err != nil {
		t.Fatalf("encode teams: %v", err)
	}
	response := &aiserverv1.GetTeamsResponse{}
	if err := proto.Unmarshal(encoded, response); err != nil {
		t.Fatalf("decode teams: %v", err)
	}
	if len(response.Teams) != 1 || !response.Teams[0].GetHasBilling() || response.Teams[0].GetSeats() == 0 || !response.Teams[0].GetIsEnterprise() {
		t.Fatalf("unexpected enterprise teams: %+v", response.Teams)
	}
}

func authRequestContext(method string, path string, body string, token string) *RequestContext {
	request := httptest.NewRequest(method, "http://local"+path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return &RequestContext{
		ResponseWriter: httptest.NewRecorder(),
		Request:        request,
		Method:         method,
		Headers:        request.Header.Clone(),
		RequestBody:    []byte(body),
	}
}
