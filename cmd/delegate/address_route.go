package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"gitea.local/ryan/new-delegate/endpoint"
	"gitea.local/ryan/new-delegate/link"
)

type addressRouteRuntime struct {
	listen func(network, address string) (net.Listener, error)
	dial   func(context.Context, string, string) (net.Conn, error)
	drain  time.Duration
}

func productionAddressRouteRuntime() addressRouteRuntime {
	dialer := new(net.Dialer)
	return addressRouteRuntime{
		listen: net.Listen,
		dial:   dialer.DialContext,
		drain:  10 * time.Second,
	}
}

func serveAddressRoute(ctx context.Context, route endpoint.Route, runtime addressRouteRuntime) error {
	if route.Ingress.Kind != endpoint.TCPListen || route.Egress.Kind != endpoint.TCPConnect {
		return fmt.Errorf("unsupported address route %s to %s", route.Ingress.Kind, route.Egress.Kind)
	}
	if runtime.listen == nil {
		return errors.New("address route listener is required")
	}
	if runtime.dial == nil {
		return errors.New("address route dialer is required")
	}
	if runtime.drain <= 0 {
		return errors.New("address route drain timeout must be positive")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	listenAddress := net.JoinHostPort(route.Ingress.Host, strconv.Itoa(int(route.Ingress.Port)))
	listener, err := runtime.listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", listenAddress, err)
	}
	defer listener.Close()

	dialAddress := net.JoinHostPort(route.Egress.Host, strconv.Itoa(int(route.Egress.Port)))
	return link.Serve(ctx, listener, func(dialContext context.Context) (net.Conn, error) {
		return runtime.dial(dialContext, "tcp", dialAddress)
	}, runtime.drain)
}
