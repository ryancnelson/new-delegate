package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type sessionOwner interface {
	beginSessionDrain() <-chan struct{}
	forceCloseSessions()
}

// Serve runs an HTTP server until it exits or the context is cancelled. On
// cancellation it stops accepting new connections and gives active requests
// the configured time to finish.
func Serve(ctx context.Context, httpServer *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- httpServer.Serve(listener)
	}()

	var serveErr error
	serveReturned := false
	select {
	case err := <-serveDone:
		serveErr = normalizeServeError(err)
		serveReturned = true
	case <-ctx.Done():
	}

	var sessions sessionOwner
	var sessionsDrained <-chan struct{}
	if owned, ok := httpServer.Handler.(sessionOwner); ok {
		sessions = owned
		sessionsDrained = sessions.beginSessionDrain()
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shutdownErr := httpServer.Shutdown(shutdownContext)
	if sessionsDrained != nil {
		select {
		case <-sessionsDrained:
		case <-shutdownContext.Done():
			sessions.forceCloseSessions()
		}
	}
	if shutdownErr != nil {
		_ = httpServer.Close()
		if sessions != nil {
			sessions.forceCloseSessions()
		}
	}
	if !serveReturned {
		serveErr = normalizeServeError(<-serveDone)
	}
	if shutdownErr != nil {
		return fmt.Errorf("graceful shutdown: %w", shutdownErr)
	}
	return serveErr
}

// connectionSessions owns streams that net/http stops tracking after a
// handler hijacks its client connection. Beginning a drain is permanent: it
// cancels pending CONNECT dials and rejects every later registration.
type connectionSessions struct {
	mu       sync.Mutex
	draining bool
	context  context.Context
	cancel   context.CancelFunc
	drained  chan struct{}
	items    map[net.Conn]struct{}
}

func newConnectionSessions() *connectionSessions {
	ctx, cancel := context.WithCancel(context.Background())
	return &connectionSessions{
		context: ctx,
		cancel:  cancel,
		drained: make(chan struct{}),
		items:   make(map[net.Conn]struct{}),
	}
}

func (s *connectionSessions) dialContext(request context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(s.context)
	stopRequestCancel := context.AfterFunc(request, cancel)
	return ctx, func() {
		stopRequestCancel()
		cancel()
	}
}

func (s *connectionSessions) add(connection net.Conn) bool {
	if connection == nil {
		return false
	}
	s.mu.Lock()
	if s.draining {
		s.mu.Unlock()
		_ = connection.Close()
		return false
	}
	s.items[connection] = struct{}{}
	s.mu.Unlock()
	return true
}

func (s *connectionSessions) remove(connection net.Conn) {
	s.mu.Lock()
	delete(s.items, connection)
	if s.draining && len(s.items) == 0 {
		s.closeDrainedLocked()
	}
	s.mu.Unlock()
}

func (s *connectionSessions) beginDrain() <-chan struct{} {
	s.mu.Lock()
	if !s.draining {
		s.draining = true
		s.cancel()
		if len(s.items) == 0 {
			s.closeDrainedLocked()
		}
	}
	drained := s.drained
	s.mu.Unlock()
	return drained
}

func (s *connectionSessions) forceClose() {
	s.mu.Lock()
	s.draining = true
	s.cancel()
	items := make([]net.Conn, 0, len(s.items))
	for connection := range s.items {
		items = append(items, connection)
		delete(s.items, connection)
	}
	s.closeDrainedLocked()
	s.mu.Unlock()

	for _, connection := range items {
		_ = connection.Close()
	}
}

func (s *connectionSessions) closeDrainedLocked() {
	select {
	case <-s.drained:
	default:
		close(s.drained)
	}
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
