// Package link provides the transport-neutral local side of an ephemeral
// geographic TCP bridge.
package link

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// Dialer opens one remote stream for an accepted local client. A future
// Tailcat adapter will implement this with a reusable in-memory pairing code.
type Dialer func(context.Context) (net.Conn, error)

// Serve accepts local TCP connections and relays each one to a remote stream
// returned by dial. On cancellation it stops accepting immediately, then waits
// for active streams to finish until drain elapses before closing them.
func Serve(ctx context.Context, listener net.Listener, dial Dialer, drain time.Duration) error {
	if listener == nil {
		return errors.New("link listener is required")
	}
	if dial == nil {
		return errors.New("link dialer is required")
	}
	if drain <= 0 {
		return errors.New("link drain timeout must be positive")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	active := newConnections()
	var streams sync.WaitGroup
	stopWatcher := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-stopWatcher:
		}
	}()
	defer close(stopWatcher)

	for {
		client, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				break
			}
			active.Close()
			streams.Wait()
			return err
		}
		if ctx.Err() != nil {
			_ = client.Close()
			break
		}
		if !active.Add(client) {
			break
		}

		streams.Add(1)
		go func(client net.Conn) {
			defer streams.Done()
			defer client.Close()
			defer active.Remove(client)

			remote, err := dial(ctx)
			if err != nil {
				return
			}
			defer remote.Close()
			if ctx.Err() != nil {
				return
			}

			if !active.Add(remote) {
				return
			}
			defer active.Remove(remote)
			relay(client, remote)
		}(client)
	}

	drained := make(chan struct{})
	go func() {
		streams.Wait()
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(drain):
		active.Close()
		<-drained
	}
	return nil
}

func relay(left, right net.Conn) {
	var copies sync.WaitGroup
	copies.Add(2)
	go copyHalf(&copies, right, left)
	go copyHalf(&copies, left, right)
	copies.Wait()
}

func copyHalf(group *sync.WaitGroup, destination, source net.Conn) {
	defer group.Done()
	_, _ = io.Copy(destination, source)
	if closeWriter, ok := destination.(interface{ CloseWrite() error }); ok {
		_ = closeWriter.CloseWrite()
	}
}

type connections struct {
	mu      sync.Mutex
	closing bool
	items   map[net.Conn]struct{}
}

func newConnections() *connections {
	return &connections{items: make(map[net.Conn]struct{})}
}

func (c *connections) Add(connection net.Conn) bool {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		_ = connection.Close()
		return false
	}
	c.items[connection] = struct{}{}
	c.mu.Unlock()
	return true
}

func (c *connections) Remove(connection net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, connection)
}

func (c *connections) Close() {
	c.mu.Lock()
	c.closing = true
	items := make([]net.Conn, 0, len(c.items))
	for connection := range c.items {
		items = append(items, connection)
	}
	c.mu.Unlock()

	for _, connection := range items {
		_ = connection.Close()
	}
}
