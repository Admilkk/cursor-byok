package backend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"cursor/internal/backend/server"
	serverconfig "cursor/internal/backend/server/config"
)

func TestHostForwardsUnhandledRoutesToOriginalUpstream(t *testing.T) {
	var requestCount atomic.Int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read upstream request body: %v", err)
		}
		writer.Header().Set("X-Upstream-Path", request.URL.RequestURI())
		writer.WriteHeader(http.StatusMultiStatus)
		_, _ = writer.Write(body)
	}))
	defer upstreamServer.Close()

	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	host, err := NewHost(store)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}

	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "managed skills", path: "/aiserver.v1.DashboardService/GetManagedSkills?source=skills"},
		{name: "effective plugins", path: "/aiserver.v1.DashboardService/GetEffectiveUserPlugins?source=plugins"},
		{name: "MCP registry", path: "/aiserver.v1.MCPRegistryService/GetKnownServers?source=mcp"},
		{name: "auth poll", path: "/auth/poll?uuid=local-login&verifier=test"},
		{name: "OAuth token", path: "/oauth/token"},
		{name: "auth email", path: "/aiserver.v1.AuthService/GetEmail"},
		{name: "dashboard me", path: "/aiserver.v1.DashboardService/GetMe"},
		{name: "full stripe profile", method: http.MethodGet, path: "/auth/full_stripe_profile"},
		{name: "stripe profile", method: http.MethodGet, path: "/auth/stripe_profile"},
		{name: "valid payment method", method: http.MethodGet, path: "/auth/has_valid_payment_method"},
		{name: "auth logout", path: "/auth/logout"},
		{name: "dashboard global commands", path: "/aiserver.v1.DashboardService/GetGlobalCommands"},
		{name: "dashboard CLI download", path: "/aiserver.v1.DashboardService/GetCliDownloadUrl"},
		{name: "dashboard privacy mode", path: "/aiserver.v1.DashboardService/GetUserPrivacyMode"},
		{name: "service catch-all", path: "/aiserver.v1.NetworkService/UnknownProcedure?source=network"},
		{name: "AI handler miss", path: "/aiserver.v1.AiService/UnknownProcedure?source=ai"},
		{name: "global miss", path: "/unknown/service/path?source=global"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			method := testCase.method
			if method == "" {
				method = http.MethodPost
			}
			body := "payload-" + testCase.name
			request := httptest.NewRequest(method, "http://localhost:8000"+testCase.path, strings.NewReader(body))
			request.Header.Set(server.HeaderServerUpstreamURL, upstreamServer.URL+testCase.path)
			recorder := httptest.NewRecorder()

			host.mux.ServeHTTP(recorder, request)

			if got := recorder.Code; got != http.StatusMultiStatus {
				t.Fatalf("status: got %d, want %d; body=%s", got, http.StatusMultiStatus, recorder.Body.String())
			}
			if got := recorder.Header().Get("X-Upstream-Path"); got != testCase.path {
				t.Fatalf("upstream path: got %q, want %q", got, testCase.path)
			}
			wantBody := body
			if method == http.MethodGet {
				wantBody = ""
			}
			if got := recorder.Body.String(); got != wantBody {
				t.Fatalf("response body: got %q, want %q", got, wantBody)
			}
		})
	}

	requestsBeforeHealthCheck := requestCount.Load()
	healthRequest := httptest.NewRequest(http.MethodGet, "http://localhost:8000"+healthPath, nil)
	healthRecorder := httptest.NewRecorder()
	host.mux.ServeHTTP(healthRecorder, healthRequest)
	if got := healthRecorder.Code; got != http.StatusOK {
		t.Fatalf("health status: got %d, want %d", got, http.StatusOK)
	}
	if got := requestCount.Load(); got != requestsBeforeHealthCheck {
		t.Fatalf("local health route unexpectedly reached upstream: requests before=%d after=%d", requestsBeforeHealthCheck, got)
	}
}

func TestHostFallbackKeepsWildcardCORSWhenUpstreamReturnsCORSHeaders(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "vscode-file://vscode-app")
		writer.Header().Set("Access-Control-Allow-Credentials", "true")
		writer.WriteHeader(http.StatusOK)
	}))
	defer upstreamServer.Close()

	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	host, err := NewHost(store)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://localhost:8000/auth/poll?uuid=test", nil)
	request.Header.Set("Origin", "vscode-file://vscode-app")
	request.Header.Set(server.HeaderServerUpstreamURL, upstreamServer.URL+request.URL.RequestURI())
	recorder := httptest.NewRecorder()

	host.mux.ServeHTTP(recorder, request)

	if got := recorder.Header().Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != "*" {
		t.Fatalf("allow origin values: got %q, want [*]", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow credentials: got %q, want empty", got)
	}
}
