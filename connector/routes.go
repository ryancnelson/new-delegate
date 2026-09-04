package connector

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

// HTTPRoutes selects a preloaded HTTP transport from the resolved mount. The
// semantic Fetch remains transport-agnostic.
type HTTPRoutes struct {
	defaultConnector *HTTP
	ftpConnector     *FTP
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
		ftpConnector:     NewFTP(nil),
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
			CheckRedirect: stopRedirects,
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
	selected, err := r.connectorForMount(mapping)
	if err != nil {
		return operation.Result{}, err
	}
	return selected.Fetch(ctx, fetch)
}

// ListForMount executes a List with the transport associated with the
// resolver-selected mount.
func (r *HTTPRoutes) ListForMount(ctx context.Context, mapping mount.Mount, list operation.List) (operation.Result, error) {
	selected, err := r.connectorForMount(mapping)
	if err != nil {
		return operation.Result{}, err
	}
	return selected.List(ctx, list)
}

// StoreForMount executes a Store with the transport associated with the
// resolver-selected mount.
func (r *HTTPRoutes) StoreForMount(ctx context.Context, mapping mount.Mount, store operation.Store) (operation.Result, error) {
	selected, err := r.connectorForMount(mapping)
	if err != nil {
		return operation.Result{}, err
	}
	return selected.Store(ctx, store)
}

type protocolConnector interface {
	Fetch(context.Context, operation.Fetch) (operation.Result, error)
	List(context.Context, operation.List) (operation.Result, error)
	Store(context.Context, operation.Store) (operation.Result, error)
}

func (r *HTTPRoutes) connectorForMount(mapping mount.Mount) (protocolConnector, error) {
	parsed, err := url.Parse(mapping.Target)
	if err != nil {
		return nil, fmt.Errorf("mount target URL: %w", err)
	}
	switch {
	case strings.EqualFold(parsed.Scheme, "ftp"):
		return r.ftpConnector, nil
	case mapping.TLS == nil:
		return r.defaultConnector, nil
	default:
		selected, ok := r.secureConnectors[*mapping.TLS]
		if !ok {
			return nil, fmt.Errorf("no preloaded backend TLS transport for selected mount")
		}
		return selected, nil
	}
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
