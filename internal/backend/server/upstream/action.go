package upstream

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cursor/internal/backend/server"
)

type CompatRouteConfig struct {
	Name          string
	StatusCode    int
	MockProtoType string
	MockBuilder   func(*RequestContext) (map[string]any, error)
	ConsoleLog    bool
}

const DefaultCursorUpstreamBaseURL = "https://api2.cursor.sh:443"

func ForwardAction(deps Dependencies, cfg CompatRouteConfig) server.HandlerFunc {
	return func(ctx *server.Context) error {
		reqCtx, route, err := newCompatRouteObjects(ctx, deps, cfg)
		if err != nil {
			return err
		}
		return handleDirect(reqCtx, route)
	}
}

// FallbackForwardAction preserves an MITM request's original upstream URL. A
// native request has no original host metadata, so it is resolved against the
// configured default upstream while retaining its path and query string.
func FallbackForwardAction(deps Dependencies, cfg CompatRouteConfig, defaultBaseURL string) server.HandlerFunc {
	forward := ForwardAction(deps, cfg)
	return func(ctx *server.Context) error {
		if ctx == nil || ctx.Request == nil || ctx.Request.URL == nil {
			return fmt.Errorf("fallback upstream request context is invalid")
		}
		if ctx.UpstreamURL == nil {
			baseURL, err := ParseAndValidateRawURL(defaultBaseURL)
			if err != nil {
				return fmt.Errorf("parse fallback upstream URL: %w", err)
			}
			targetURL := *ctx.Request.URL
			targetURL.Scheme = baseURL.Scheme
			targetURL.Host = baseURL.Host
			targetURL.User = baseURL.User
			ctx.UpstreamURL = &targetURL
		}
		return forward(ctx)
	}
}

func FixedStatusAction(deps Dependencies, cfg CompatRouteConfig) server.HandlerFunc {
	return func(ctx *server.Context) error {
		reqCtx, route, err := newCompatRouteObjects(ctx, deps, cfg)
		if err != nil {
			return err
		}
		return handleFixedStatus(reqCtx, route)
	}
}

func MockDevSessionTokenAction(deps Dependencies, cfg CompatRouteConfig) server.HandlerFunc {
	return func(ctx *server.Context) error {
		reqCtx, route, err := newCompatRouteObjects(ctx, deps, cfg)
		if err != nil {
			return err
		}
		return handleMockDevSessionToken(reqCtx, route)
	}
}

func MockProtoAction(deps Dependencies, cfg CompatRouteConfig) server.HandlerFunc {
	return func(ctx *server.Context) error {
		reqCtx, route, err := newCompatRouteObjects(ctx, deps, cfg)
		if err != nil {
			return err
		}
		return handleMockProto(reqCtx, route)
	}
}

func newCompatRouteObjects(ctx *server.Context, deps Dependencies, cfg CompatRouteConfig) (*RequestContext, *Route, error) {
	if ctx == nil || ctx.Request == nil {
		return nil, nil, nil
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, nil, err
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	targetURL := ctx.UpstreamURL
	if targetURL == nil && ctx.Request.URL != nil {
		copyURL := *ctx.Request.URL
		targetURL = &copyURL
	}
	reqCtx := &RequestContext{
		ResponseWriter: ctx.Writer,
		Request:        ctx.Request,
		StartedAt:      ctx.StartedAt,
		RawURL:         strings.TrimSpace(ctx.Request.Header.Get(server.HeaderServerUpstreamURL)),
		TargetURL:      targetURL,
		Method:         strings.ToUpper(strings.TrimSpace(ctx.Request.Method)),
		Headers:        ctx.Request.Header.Clone(),
		ContentType:    strings.TrimSpace(ctx.Request.Header.Get("content-type")),
		RequestBody:    body,
		Deps:           &deps,
		HTTPRequestID:  resolveHTTPRequestID(ctx.Request),
	}
	route := &Route{
		Name:               cfg.Name,
		Pattern:            ctx.Request.URL.Path,
		StatusCode:         cfg.StatusCode,
		MockProtoType:      cfg.MockProtoType,
		MockPayloadBuilder: cfg.MockBuilder,
		ConsoleLog:         cfg.ConsoleLog,
	}
	return reqCtx, route, nil
}

func ServerTimeMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildServerTimePayload(reqCtx)
}

func ServerConfigMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildServerConfigPayload(reqCtx)
}

func AvailableModelsMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildAvailableModelsPayload(reqCtx)
}

func DefaultModelNudgeMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDefaultModelNudgeDataPayload(reqCtx)
}

func UsableModelsMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildUsableModelsPayload(reqCtx)
}

func DefaultModelForCliMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDefaultModelForCliPayload(reqCtx)
}

func DefaultModelMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDefaultModelPayload(reqCtx)
}

func BootstrapStatsigMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildBootstrapStatsigPayload(reqCtx)
}

func FirstWindowStatsigDecisionMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildFirstWindowStatsigDecisionPayload(reqCtx)
}

func DashboardCurrentPeriodUsageMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDashboardCurrentPeriodUsagePayload(reqCtx)
}

func DashboardTeamsMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDashboardTeamsPayload(reqCtx)
}

// EmptyMockBuilder возвращает пустой proto-ответ для ручек, где клиенту
// достаточно успешного "пусто": нет team-настроек, нет репозиториев,
// нет маркетплейсов/плагинов/команд, телеметрия принята без обработки.
func EmptyMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return map[string]any{}, nil
}

// SubmitLogsMockBuilder подтверждает приём логов телеметрии без обработки.
func SubmitLogsMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return map[string]any{"success": true}, nil
}

func DashboardPlanInfoMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDashboardPlanInfoPayload(reqCtx)
}

func DashboardUsageLimitStatusAndActiveGrantsMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDashboardUsageLimitStatusAndActiveGrantsPayload(reqCtx)
}

func DashboardIsOnNewPricingMockBuilder(reqCtx *RequestContext) (map[string]any, error) {
	return buildDashboardIsOnNewPricingPayload(reqCtx)
}

func resolveHTTPRequestID(request *http.Request) string {
	requestID := strings.TrimSpace(request.Header.Get("x-request-id"))
	if requestID != "" {
		return requestID
	}
	return strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-")
}
