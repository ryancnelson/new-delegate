package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/connector"
	gatewayserver "gitea.local/ryan/new-delegate/server"
	"gitea.local/ryan/new-delegate/tlsconfig"
	"gitea.local/ryan/new-delegate/tlsruntime"
)

type listenFunction func(network, address string) (net.Listener, error)

// prepareHTTPRuntime loads every frontend identity before binding any socket,
// then returns a complete plaintext/TLS listener group ready to serve.
func prepareHTTPRuntime(
	configured config.Config,
	snapshot func() config.Config,
	backendClient *http.Client,
	listen listenFunction,
) ([]*http.Server, []net.Listener, *connector.HTTPRoutes, error) {
	backendPolicies := make(map[tlsconfig.Backend]*tls.Config)
	for _, mounted := range configured.Mounts {
		if mounted.TLS == nil {
			continue
		}
		policy := *mounted.TLS
		if _, loaded := backendPolicies[policy]; loaded {
			continue
		}
		tlsConfig, err := tlsruntime.LoadBackend(policy)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load backend TLS policy: %w", err)
		}
		backendPolicies[policy] = tlsConfig
	}
	routes := connector.NewHTTPRoutes(backendClient, backendPolicies)

	tlsConfigs := make([]*tls.Config, len(configured.Servers))
	for i, frontend := range configured.Servers {
		if frontend.TLS == nil {
			continue
		}
		loaded, err := tlsruntime.LoadFrontend(*frontend.TLS)
		if err != nil {
			routes.CloseIdleConnections()
			return nil, nil, nil, fmt.Errorf("load TLS for server %q: %w", frontend.Name, err)
		}
		tlsConfigs[i] = loaded
	}

	httpServers := make([]*http.Server, 0, len(configured.Servers))
	listeners := make([]net.Listener, 0, len(configured.Servers))
	for i, frontend := range configured.Servers {
		handler := gatewayserver.NewReloadableHTTPHandlerWithRoutes(frontend.Name, snapshot, routes)
		httpServer := &http.Server{
			Addr:              frontend.Listen,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}
		listener, err := listen("tcp", httpServer.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			routes.CloseIdleConnections()
			return nil, nil, nil, fmt.Errorf("listen for server %q: %w", frontend.Name, err)
		}
		if tlsConfigs[i] != nil {
			listener = tls.NewListener(listener, tlsConfigs[i])
		}
		httpServers = append(httpServers, httpServer)
		listeners = append(listeners, listener)
	}
	return httpServers, listeners, routes, nil
}
