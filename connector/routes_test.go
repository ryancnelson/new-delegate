package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestHTTPRoutesSelectsTransportFromMountPolicy(t *testing.T) {
	plainCalls := atomic.Int32{}
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainCalls.Add(1)
		_, _ = io.WriteString(w, "plain")
	}))
	defer plain.Close()
	secureCalls := atomic.Int32{}
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secureCalls.Add(1)
		_, _ = io.WriteString(w, "secure")
	}))
	defer secure.Close()

	policy := tlsconfig.Backend{ServerName: "example.com", MinimumVersion: "1.2"}
	roots := x509.NewCertPool()
	roots.AddCert(secure.Certificate())
	routes := NewHTTPRoutes(&http.Client{}, map[tlsconfig.Backend]*tls.Config{
		policy: {RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12},
	})
	defer routes.CloseIdleConnections()

	for _, tt := range []struct {
		name    string
		mapping mount.Mount
		url     string
		body    string
	}{
		{name: "default plaintext", mapping: mount.Mount{}, url: plain.URL, body: "plain"},
		{name: "selected TLS policy", mapping: mount.Mount{TLS: &policy}, url: secure.URL, body: "secure"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routes.FetchForMount(context.Background(), tt.mapping, operation.Fetch{
				Method: http.MethodGet, Resource: tt.url,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			if err != nil || string(body) != tt.body {
				t.Fatalf("body = %q, %v; want %q", body, err, tt.body)
			}
		})
	}

	unknown := tlsconfig.Backend{ServerName: "unknown.internal"}
	_, err := routes.FetchForMount(context.Background(), mount.Mount{TLS: &unknown}, operation.Fetch{
		Method: http.MethodGet, Resource: secure.URL,
	})
	if err == nil || !strings.Contains(err.Error(), "no preloaded backend TLS transport") {
		t.Fatalf("unknown route error = %v", err)
	}
	if plainCalls.Load() != 1 || secureCalls.Load() != 1 {
		t.Fatalf("backend calls = plain %d, secure %d; unknown route must not connect", plainCalls.Load(), secureCalls.Load())
	}
}

func TestHTTPRoutesExecutesStore(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if request.Method != http.MethodPut || request.URL.Path != "/objects/report" || string(body) != "contents" || request.ContentLength != 8 {
			t.Errorf("request = %s %s length=%d body=%q", request.Method, request.URL.Path, request.ContentLength, body)
		}
		response.WriteHeader(http.StatusCreated)
	}))
	defer backend.Close()
	routes := NewHTTPRoutes(backend.Client(), nil)
	result, err := routes.StoreForMount(context.Background(), mount.Mount{}, operation.Store{
		Method: http.MethodPut, Resource: backend.URL + "/objects/report",
		Body: strings.NewReader("contents"), Size: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if result.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", result.Status, http.StatusCreated)
	}
}
