package upstream

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	legacyruntime "cursor/internal/runtime"
)

const (
	localDevDefaultPlan        = "ultra"
	localDevTokenLifetime      = 10 * 365 * 24 * time.Hour
	localDevSubscriptionActive = "active"
)

var localDevPlans = map[string]struct{}{
	"free":       {},
	"pro":        {},
	"pro_plus":   {},
	"ultra":      {},
	"enterprise": {},
}

type localDevSessionClaims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	Plan      string `json:"cursor_local_plan"`
	Trial     bool   `json:"cursor_local_trial"`
	TokenType string `json:"type"`
	Issuer    string `json:"iss"`
	Scope     string `json:"scope"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func handleMockDevSessionToken(reqCtx *RequestContext, route *Route) error {
	_ = route
	if reqCtx == nil || reqCtx.Request == nil || reqCtx.ResponseWriter == nil {
		return fmt.Errorf("dev session request context is invalid")
	}

	plan, trial, email, err := parseLocalDevSessionQuery(reqCtx.Request)
	if err != nil {
		writeJSONError(reqCtx.ResponseWriter, http.StatusBadRequest, err.Error())
		return nil
	}

	token, claims, err := buildLocalDevSessionToken(plan, trial, email, time.Now())
	if err != nil {
		return err
	}
	responseBody, err := marshalJSONBody(map[string]any{
		"accessToken":  token,
		"refreshToken": token,
		"authId":       claims.Subject,
	})
	if err != nil {
		return err
	}
	reqCtx.ResponseWriter.Header().Set("content-type", "application/json")
	reqCtx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = reqCtx.ResponseWriter.Write(responseBody)
	return nil
}

func parseLocalDevSessionQuery(request *http.Request) (string, bool, string, error) {
	plan := localDevDefaultPlan
	email := legacyruntime.InjectAccountEmail
	if request == nil || request.URL == nil {
		return plan, false, email, nil
	}

	query := request.URL.Query()
	if requestedPlan := strings.TrimSpace(query.Get("plan")); requestedPlan != "" {
		plan = requestedPlan
	}
	if _, ok := localDevPlans[plan]; !ok {
		return "", false, "", fmt.Errorf("unsupported dev plan %q", plan)
	}

	trial := false
	if rawTrial := strings.TrimSpace(query.Get("trial")); rawTrial != "" {
		parsed, err := strconv.ParseBool(rawTrial)
		if err != nil {
			return "", false, "", fmt.Errorf("invalid trial value %q", rawTrial)
		}
		trial = parsed
	}
	if trial && plan != "pro" && plan != "pro_plus" {
		return "", false, "", fmt.Errorf("trial is only supported for pro and pro_plus")
	}

	if requestedEmail := strings.TrimSpace(query.Get("email")); requestedEmail != "" {
		email = requestedEmail
	}
	return plan, trial, email, nil
}

func buildLocalDevSessionToken(plan string, trial bool, email string, now time.Time) (string, localDevSessionClaims, error) {
	authID := "local-dev-" + strings.ReplaceAll(plan, "_", "-")
	if trial {
		authID += "-trial"
	}
	claims := localDevSessionClaims{
		Subject:   authID,
		Email:     strings.TrimSpace(email),
		Plan:      plan,
		Trial:     trial,
		TokenType: "session",
		Issuer:    "cursor-local-backend",
		Scope:     "openid profile email",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(localDevTokenLifetime).Unix(),
	}
	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", localDevSessionClaims{}, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", localDevSessionClaims{}, err
	}
	encode := base64.RawURLEncoding.EncodeToString
	token := encode(headerJSON) + "." + encode(claimsJSON) + ".local-dev"
	return token, claims, nil
}

func localDevClaimsFromRequest(reqCtx *RequestContext) (localDevSessionClaims, bool) {
	if reqCtx == nil {
		return localDevSessionClaims{}, false
	}
	return localDevClaimsFromAuthorization(reqCtx.Headers.Get("authorization"))
}

func localDevClaimsFromAuthorization(authorization string) (localDevSessionClaims, bool) {
	authorization = strings.TrimSpace(authorization)
	if len(authorization) >= len("Bearer ") && strings.EqualFold(authorization[:len("Bearer ")], "Bearer ") {
		authorization = strings.TrimSpace(authorization[len("Bearer "):])
	}
	parts := strings.Split(authorization, ".")
	if len(parts) != 3 {
		return localDevSessionClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return localDevSessionClaims{}, false
	}
	claims := localDevSessionClaims{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return localDevSessionClaims{}, false
	}
	if claims.Issuer != "cursor-local-backend" {
		return localDevSessionClaims{}, false
	}
	if _, ok := localDevPlans[claims.Plan]; !ok || strings.TrimSpace(claims.Subject) == "" {
		return localDevSessionClaims{}, false
	}
	return claims, true
}

func localDevPlanFromRequest(reqCtx *RequestContext) string {
	if claims, ok := localDevClaimsFromRequest(reqCtx); ok {
		return claims.Plan
	}
	return localDevDefaultPlan
}

func localDevPlanDetails(plan string) (string, int) {
	switch plan {
	case "free":
		return "Free Plan", 0
	case "pro":
		return "Pro Plan", 2000
	case "pro_plus":
		return "Pro+ Plan", 6000
	case "enterprise":
		return "Enterprise Plan", 0
	default:
		return "Ultra Plan", localUltraPlanIncludedCents
	}
}

func writeJSONError(writer http.ResponseWriter, statusCode int, message string) {
	writer.Header().Set("content-type", "application/json")
	writer.WriteHeader(statusCode)
	payload, _ := json.Marshal(map[string]string{"error": message})
	_, _ = writer.Write(payload)
}
