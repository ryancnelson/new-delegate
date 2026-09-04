package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/connector"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/policy"
	"gitea.local/ryan/new-delegate/tlsconfig"
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

func TestHTTPHandlerRoutesFetchWithSelectedMount(t *testing.T) {
	tlsPolicy := &tlsconfig.Backend{CAFile: "ca.pem", ServerName: "backend.internal"}
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts: []mount.Mount{{
			Path: "/secure/*", Target: "https://backend.internal/*", TLS: tlsPolicy,
		}},
		Policies: []policy.Rule{{
			Effect: policy.Permit, Protocol: "http", Destination: "backend.internal", Source: "*",
		}},
	}
	router := &recordingMountConnector{}
	handler := NewReloadableHTTPHandlerWithRoutes("public", func() config.Config { return configured }, router)
	request := httptest.NewRequest(http.MethodGet, "http://frontend/secure/report", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || router.calls != 1 {
		t.Fatalf("response = %d, calls=%d", response.Code, router.calls)
	}
	if router.mapping.TLS == nil || *router.mapping.TLS != *tlsPolicy {
		t.Fatalf("routed mount = %#v, want selected TLS policy", router.mapping)
	}
	if router.fetch.Resource != "https://backend.internal/report" {
		t.Fatalf("Fetch resource = %q", router.fetch.Resource)
	}
}

type recordingMountConnector struct {
	calls   int
	mapping mount.Mount
	fetch   operation.Fetch
}

func (r *recordingMountConnector) FetchForMount(_ context.Context, mapping mount.Mount, fetch operation.Fetch) (operation.Result, error) {
	r.calls++
	r.mapping = mapping
	r.fetch = fetch
	return operation.Result{Status: http.StatusNoContent}, nil
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

func TestHTTPHandlerReloadsRoutingAndPolicyFromOneSnapshot(t *testing.T) {
	oldBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "old")
	}))
	defer oldBackend.Close()
	newBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "new")
	}))
	defer newBackend.Close()

	makeConfig := func(target string) config.Config {
		return config.Config{
			Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
			Mounts:  []mount.Mount{{Path: "/*", Target: target + "/*", Server: "public"}},
			Policies: []policy.Rule{{
				Effect: policy.Permit, Protocol: "http", Destination: "*",
			}},
		}
	}
	store, err := config.NewStore(makeConfig(oldBackend.URL))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewReloadableHTTPHandler("public", store.Snapshot, connector.NewHTTP(oldBackend.Client()))

	request := httptest.NewRequest(http.MethodGet, "http://frontend/resource", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "old" {
		t.Fatalf("before reload = %d %q, want old backend", response.Code, response.Body.String())
	}

	if err := store.Replace(makeConfig(newBackend.URL)); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "http://frontend/resource", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "new" {
		t.Fatalf("after reload = %d %q, want new backend", response.Code, response.Body.String())
	}
}

func TestHTTPHandlerTrustsForwardedClientOnlyFromConfiguredProxy(t *testing.T) {
	backendCalls := atomic.Int32{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		backendCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()
	configured := config.Config{
		Servers: []config.Server{{
			Name: "public", Protocol: "http", Listen: ":8080",
			ClientIPHeader: "X-Forwarded-For", TrustedProxies: []string{"10.0.0.0/8"},
		}},
		Mounts:   []mount.Mount{{Path: "/*", Target: backend.URL + "/*"}},
		Policies: []policy.Rule{{Effect: policy.Permit, Protocol: "http", Source: "203.0.113.7"}},
	}
	store, err := config.NewStore(configured)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewReloadableHTTPHandler("public", store.Snapshot, connector.NewHTTP(backend.Client()))

	for _, tt := range []struct {
		name       string
		remote     string
		forwarded  string
		wantStatus int
	}{
		{name: "trusted proxy", remote: "10.0.0.2:1234", forwarded: "203.0.113.7", wantStatus: http.StatusNoContent},
		{name: "untrusted peer cannot spoof", remote: "192.0.2.2:1234", forwarded: "203.0.113.7", wantStatus: http.StatusForbidden},
		{name: "malformed trusted chain", remote: "10.0.0.2:1234", forwarded: "not-an-address", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://frontend/resource", nil)
			request.RemoteAddr = tt.remote
			request.Header.Set("X-Forwarded-For", tt.forwarded)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
	if got := backendCalls.Load(); got != 1 {
		t.Fatalf("backend calls = %d, want exactly one trusted request", got)
	}
}
