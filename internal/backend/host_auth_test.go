package backend

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"cursor/gen/aiserverv1"
	serverconfig "cursor/internal/backend/server/config"
	"cursor/internal/certs"

	"google.golang.org/protobuf/proto"
)

func TestHostServesDevLoginAndLocalTeamsRoute(t *testing.T) {
	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	host, err := NewHost(store)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "http://local/auth/cursor_dev_session_token?plan=enterprise&email=enterprise%40example.com", nil)
	loginRecorder := httptest.NewRecorder()
	host.mux.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("dev login status: got %d, want %d; body=%s", loginRecorder.Code, http.StatusOK, loginRecorder.Body.String())
	}
	var loginResponse struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginResponse); err != nil {
		t.Fatalf("decode dev login: %v", err)
	}
	if loginResponse.AccessToken == "" {
		t.Fatal("dev login returned an empty access token")
	}

	teamsRequest := httptest.NewRequest(http.MethodPost, "http://local/aiserver.v1.DashboardService/GetTeams", nil)
	teamsRequest.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
	teamsRecorder := httptest.NewRecorder()
	host.mux.ServeHTTP(teamsRecorder, teamsRequest)
	if teamsRecorder.Code != http.StatusOK {
		t.Fatalf("teams status: got %d, want %d", teamsRecorder.Code, http.StatusOK)
	}
	teams := &aiserverv1.GetTeamsResponse{}
	if err := proto.Unmarshal(teamsRecorder.Body.Bytes(), teams); err != nil {
		t.Fatalf("decode teams response: %v", err)
	}
	if len(teams.GetTeams()) != 1 || !teams.GetTeams()[0].GetIsEnterprise() {
		t.Fatalf("unexpected teams response: %v", teams.GetTeams())
	}
}

func TestHostAllowsWildcardCORS(t *testing.T) {
	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	host, err := NewHost(store)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "http://local/auth/cursor_dev_session_token?plan=free", nil)
	preflightRequest.Header.Set("Origin", "vscode-file://vscode-app")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightRequest.Header.Set("Access-Control-Request-Headers", "x-cursor-client-type")
	preflightRecorder := httptest.NewRecorder()
	host.mux.ServeHTTP(preflightRecorder, preflightRequest)
	if preflightRecorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status: got %d, want %d", preflightRecorder.Code, http.StatusNoContent)
	}
	if got := preflightRecorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("preflight allow origin: got %q", got)
	}
	if got := preflightRecorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("preflight allow credentials: got %q, want empty", got)
	}
	if got := preflightRecorder.Header().Get("Access-Control-Allow-Headers"); got != "x-cursor-client-type" {
		t.Fatalf("preflight allow headers: got %q", got)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "http://local/auth/cursor_dev_session_token?plan=free", nil)
	loginRequest.Header.Set("Origin", "vscode-file://vscode-app")
	loginRequest.Header.Set("x-cursor-client-type", "ide")
	loginRecorder := httptest.NewRecorder()
	host.mux.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("dev login status: got %d, want %d", loginRecorder.Code, http.StatusOK)
	}
	if got := loginRecorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("dev login allow origin: got %q", got)
	}
}

func TestHostAllowsRemoteWebOriginWithWildcard(t *testing.T) {
	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	host, err := NewHost(store)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}

	request := httptest.NewRequest(http.MethodOptions, "http://local/auth/cursor_dev_session_token", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()
	host.mux.ServeHTTP(recorder, request)
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("remote origin allow origin: got %q, want wildcard", got)
	}
}

func TestHostServesDevLoginOverTrustedLocalhostTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve backend port: %v", err)
	}
	listenAddr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release backend port: %v", err)
	}

	store := serverconfig.NewStore(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir())
	config := serverconfig.DefaultConfig()
	config.BackendListenAddr = listenAddr
	if _, err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save backend config: %v", err)
	}
	certificateManager, err := certs.NewEmbeddedManager()
	if err != nil {
		t.Fatalf("new certificate manager: %v", err)
	}
	serverCertificate, err := certificateManager.CertificateForServerName("localhost")
	if err != nil {
		t.Fatalf("create localhost certificate: %v", err)
	}
	host, err := NewHost(store, WithTLSCertificate(serverCertificate))
	if err != nil {
		t.Fatalf("new TLS host: %v", err)
	}
	if err := host.Start(); err != nil {
		t.Fatalf("start TLS host: %v", err)
	}
	defer func() {
		stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := host.Stop(stopContext); err != nil {
			t.Errorf("stop TLS host: %v", err)
		}
	}()

	caCertificate, err := certificateManager.CATLSCertificate()
	if err != nil {
		t.Fatalf("load CA certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCertificate.Leaf)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
		ServerName: "localhost",
	}}}
	response, err := client.Get(host.BaseURL() + "/auth/cursor_dev_session_token?plan=pro&trial=true")
	if err != nil {
		t.Fatalf("request dev login over TLS: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("dev login TLS status: got %d, want %d", response.StatusCode, http.StatusOK)
	}
}
