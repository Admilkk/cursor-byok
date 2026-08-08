package upstream

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"cursor/internal/backend/server"
)

type fallbackHTTPClientFunc func(*http.Request) (*http.Response, error)

func (fn fallbackHTTPClientFunc) Do(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFallbackForwardActionUsesOriginalMITMUpstreamURL(t *testing.T) {
	originalURL := "https://api3.cursor.sh/aiserver.v1.UnknownService/Call?mode=exact"
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		t.Fatalf("parse original URL: %v", err)
	}

	client := fallbackHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.URL.String(); got != originalURL {
			t.Fatalf("upstream URL: got %q, want %q", got, originalURL)
		}
		if got := request.Method; got != http.MethodPost {
			t.Fatalf("method: got %q, want POST", got)
		}
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Fatalf("read request body: %v", readErr)
		}
		if got := string(body); got != "request-body" {
			t.Fatalf("body: got %q", got)
		}
		if got := request.Header.Get("X-Test-Header"); got != "preserved" {
			t.Fatalf("custom header: got %q", got)
		}
		if got := request.Header.Get(server.HeaderServerUpstreamURL); got != "" {
			t.Fatalf("internal upstream header leaked: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Status:     "202 Accepted",
			Header:     http.Header{"X-Upstream-Response": []string{"preserved"}},
			Body:       io.NopCloser(strings.NewReader("upstream-body")),
		}, nil
	})

	request := httptest.NewRequest(http.MethodPost, "http://localhost:8000/ignored", strings.NewReader("request-body"))
	request.Header.Set("X-Test-Header", "preserved")
	request.Header.Set(server.HeaderServerUpstreamURL, originalURL)
	recorder := httptest.NewRecorder()
	ctx := &server.Context{Writer: recorder, Request: request, UpstreamURL: parsedURL}
	action := FallbackForwardAction(Dependencies{HTTPClient: client}, CompatRouteConfig{Name: "fallback"}, DefaultCursorUpstreamBaseURL)

	if err := action(ctx); err != nil {
		t.Fatalf("forward fallback request: %v", err)
	}
	if got := recorder.Code; got != http.StatusAccepted {
		t.Fatalf("response status: got %d, want %d", got, http.StatusAccepted)
	}
	if got := recorder.Header().Get("X-Upstream-Response"); got != "preserved" {
		t.Fatalf("response header: got %q", got)
	}
	if got := recorder.Body.String(); got != "upstream-body" {
		t.Fatalf("response body: got %q", got)
	}
}

func TestFallbackForwardActionUsesDefaultUpstreamForNativeRequest(t *testing.T) {
	const defaultBaseURL = "https://fallback.example:8443"
	wantURL := defaultBaseURL + "/aiserver.v1.UnknownService/Call?mode=native"
	client := fallbackHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.URL.String(); got != wantURL {
			t.Fatalf("upstream URL: got %q, want %q", got, wantURL)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})

	request := httptest.NewRequest(http.MethodGet, "http://localhost:8000/aiserver.v1.UnknownService/Call?mode=native", nil)
	recorder := httptest.NewRecorder()
	ctx := &server.Context{Writer: recorder, Request: request}
	action := FallbackForwardAction(Dependencies{HTTPClient: client}, CompatRouteConfig{Name: "fallback"}, defaultBaseURL)

	if err := action(ctx); err != nil {
		t.Fatalf("forward fallback request: %v", err)
	}
	if got := recorder.Code; got != http.StatusNoContent {
		t.Fatalf("response status: got %d, want %d", got, http.StatusNoContent)
	}
}

func TestFallbackForwardActionPreservesAuthorization(t *testing.T) {
	const (
		originalURL           = "https://api2.cursor.sh/aiserver.v1.AuthService/GetEmail"
		officialAuthorization = "Bearer official-access-token"
		officialChecksum      = "official-checksum"
	)
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		t.Fatalf("parse original URL: %v", err)
	}

	client := fallbackHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != officialAuthorization {
			t.Fatalf("authorization: got %q, want %q", got, officialAuthorization)
		}
		if got := request.Header.Get("x-cursor-checksum"); got != officialChecksum {
			t.Fatalf("checksum: got %q, want %q", got, officialChecksum)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("upstream-account")),
		}, nil
	})

	request := httptest.NewRequest(http.MethodPost, "http://localhost:8000/aiserver.v1.AuthService/GetEmail", nil)
	request.Header.Set("Authorization", officialAuthorization)
	request.Header.Set("x-cursor-checksum", officialChecksum)
	recorder := httptest.NewRecorder()
	ctx := &server.Context{Writer: recorder, Request: request, UpstreamURL: parsedURL}
	action := FallbackForwardAction(
		Dependencies{HTTPClient: client},
		CompatRouteConfig{Name: "fallback"},
		DefaultCursorUpstreamBaseURL,
	)

	if err := action(ctx); err != nil {
		t.Fatalf("forward authenticated fallback request: %v", err)
	}
	if got := recorder.Body.String(); got != "upstream-account" {
		t.Fatalf("response body: got %q, want upstream-account", got)
	}
}
