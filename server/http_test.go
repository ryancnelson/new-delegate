package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"gitea.local/ryan/new-delegate/connector"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

func TestHTTPHandlerProxiesAuthorizedFetch(t *testing.T) {
	var backendCalls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalls.Add(1)
		if r.URL.Path != "/v1/users" || r.URL.RawQuery != "active=true" {
			t.Errorf("backend request target = %q, want /v1/users?active=true", r.URL.RequestURI())
		}
		if got := r.Header.Get("X-Request-ID"); got != "request-1" {
			t.Errorf("backend X-Request-ID = %q, want request-1", got)
		}
		w.Header().Set("X-Backend", "test")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "created")
	}))
	defer backend.Close()

	handler := NewHTTPHandler(
		[]mount.Mount{{Path: "/api/*", Target: backend.URL + "/v1/*"}},
		[]policy.Rule{{Effect: policy.Permit, Protocol: "http", Method: "GET", Mount: "/api/*"}},
		connector.NewHTTP(backend.Client()),
	)
	frontend := httptest.NewServer(handler)
	defer frontend.Close()

	req, err := http.NewRequest(http.MethodGet, frontend.URL+"/api/users?active=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID", "request-1")
	response, err := frontend.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if got := response.Header.Get("X-Backend"); got != "test" {
		t.Fatalf("X-Backend = %q, want test", got)
	}
	if string(body) != "created" {
		t.Fatalf("body = %q, want created", body)
	}
	if calls := backendCalls.Load(); calls != 1 {
		t.Fatalf("backend calls = %d, want 1", calls)
	}
}

func TestHTTPHandlerDenialNeverContactsBackend(t *testing.T) {
	var backendCalls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		backendCalls.Add(1)
	}))
	defer backend.Close()

	handler := NewHTTPHandler(
		[]mount.Mount{{Path: "/*", Target: backend.URL + "/*"}},
		nil,
		connector.NewHTTP(backend.Client()),
	)
	request := httptest.NewRequest(http.MethodGet, "http://frontend/secret", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%q", response.Code, http.StatusForbidden, response.Body.String())
	}
	if calls := backendCalls.Load(); calls != 0 {
		t.Fatalf("backend calls = %d, want zero", calls)
	}
	if !strings.Contains(response.Body.String(), string(policy.ReasonDefaultDeny)) {
		t.Fatalf("body = %q, want stable denial reason", response.Body.String())
	}
}

func TestHTTPHandlerUsesNamedServerForMountScope(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer backend.Close()

	handler := NewHTTPHandlerForServer(
		"public",
		[]mount.Mount{
			{Path: "/*", Target: "http://wrong.invalid/*", Server: "admin"},
			{Path: "/*", Target: backend.URL + "/public/*", Server: "public", Protocol: "http"},
		},
		[]policy.Rule{{Effect: policy.Permit, Protocol: "http"}},
		connector.NewHTTP(backend.Client()),
	)
	request := httptest.NewRequest(http.MethodGet, "http://frontend/docs", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "/public/docs" {
		t.Fatalf("response = %d %q, want scoped public backend", response.Code, response.Body.String())
	}
}
