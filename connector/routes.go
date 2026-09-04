package connector

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

// HTTPRoutes selects a preloaded HTTP transport from the resolved mount. The
// semantic Fetch remains transport-agnostic.
type HTTPRoutes struct {
	defaultConnector *HTTP
	secureConnectors map[tlsconfig.Backend]*HTTP
	transports       []*http.Transport
}

// NewHTTPRoutes constructs bounded standard-library transports for already
// loaded TLS policies.
func NewHTTPRoutes(defaultClient *http.Client, policies map[tlsconfig.Backend]*tls.Config) *HTTPRoutes {
	if defaultClient == nil {
		defaultClient = http.DefaultClient
	}
	routes := &HTTPRoutes{
		defaultConnector: NewHTTP(defaultClient),
		secureConnectors: make(map[tlsconfig.Backend]*HTTP, len(policies)),
		transports:       make([]*http.Transport, 0, len(policies)),
	}
	for policy, tlsConfig := range policies {
		if tlsConfig == nil {
			continue
		}
		transport := cloneHTTPTransport(defaultClient.Transport)
		transport.TLSClientConfig = tlsConfig.Clone()
		client := &http.Client{
			Transport:     transport,
			CheckRedirect: defaultClient.CheckRedirect,
			Jar:           defaultClient.Jar,
			Timeout:       defaultClient.Timeout,
		}
		routes.secureConnectors[policy] = NewHTTP(client)
		routes.transports = append(routes.transports, transport)
	}
	return routes
}

// FetchForMount executes a Fetch with the transport associated with the
// resolver-selected mount.
func (r *HTTPRoutes) FetchForMount(ctx context.Context, mapping mount.Mount, fetch operation.Fetch) (operation.Result, error) {
	if mapping.TLS == nil {
		return r.defaultConnector.Fetch(ctx, fetch)
	}
	selected, ok := r.secureConnectors[*mapping.TLS]
	if !ok {
		return operation.Result{}, fmt.Errorf("no preloaded backend TLS transport for selected mount")
	}
	return selected.Fetch(ctx, fetch)
}

// CloseIdleConnections releases pooled connections owned by routed transports.
func (r *HTTPRoutes) CloseIdleConnections() {
	if r == nil {
		return
	}
	if r.defaultConnector != nil && r.defaultConnector.client != nil {
		r.defaultConnector.client.CloseIdleConnections()
	}
	for _, transport := range r.transports {
		transport.CloseIdleConnections()
	}
}

func cloneHTTPTransport(source http.RoundTripper) *http.Transport {
	if configured, ok := source.(*http.Transport); ok {
		return configured.Clone()
	}
	return http.DefaultTransport.(*http.Transport).Clone()
}
