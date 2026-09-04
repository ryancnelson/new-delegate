package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ServeAll runs a set of HTTP listeners under one lifecycle. The first
// listener failure cancels and drains the rest; parent cancellation drains all
// listeners before returning.
func ServeAll(ctx context.Context, servers []*http.Server, listeners []net.Listener, shutdownTimeout time.Duration) error {
	if len(servers) == 0 || len(servers) != len(listeners) {
		return fmt.Errorf("servers and listeners must have the same non-zero length")
	}

	groupContext, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, len(servers))
	for i := range servers {
		server, listener := servers[i], listeners[i]
		go func() {
			results <- Serve(groupContext, server, listener, shutdownTimeout)
		}()
	}

	first := <-results
	parentStopped := ctx.Err() != nil
	cancel()
	result := first
	for range len(servers) - 1 {
		if err := <-results; result == nil && err != nil {
			result = err
		}
	}
	if result == nil && !parentStopped {
		return fmt.Errorf("listener stopped unexpectedly")
	}
	return result
}
