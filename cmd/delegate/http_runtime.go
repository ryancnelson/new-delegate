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
	"gitea.local/ryan/new-delegate/tlsruntime"
)

type listenFunction func(network, address string) (net.Listener, error)

// prepareHTTPRuntime loads every frontend identity before binding any socket,
// then returns a complete plaintext/TLS listener group ready to serve.
func prepareHTTPRuntime(
	configured config.Config,
	snapshot func() config.Config,
	backend *connector.HTTP,
	listen listenFunction,
) ([]*http.Server, []net.Listener, error) {
	tlsConfigs := make([]*tls.Config, len(configured.Servers))
	for i, frontend := range configured.Servers {
		if frontend.TLS == nil {
			continue
		}
		loaded, err := tlsruntime.LoadFrontend(*frontend.TLS)
		if err != nil {
			return nil, nil, fmt.Errorf("load TLS for server %q: %w", frontend.Name, err)
		}
		tlsConfigs[i] = loaded
	}

	httpServers := make([]*http.Server, 0, len(configured.Servers))
	listeners := make([]net.Listener, 0, len(configured.Servers))
	for i, frontend := range configured.Servers {
		handler := gatewayserver.NewReloadableHTTPHandler(frontend.Name, snapshot, backend)
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
			return nil, nil, fmt.Errorf("listen for server %q: %w", frontend.Name, err)
		}
		if tlsConfigs[i] != nil {
			listener = tls.NewListener(listener, tlsConfigs[i])
		}
		httpServers = append(httpServers, httpServer)
		listeners = append(listeners, listener)
	}
	return httpServers, listeners, nil
}
