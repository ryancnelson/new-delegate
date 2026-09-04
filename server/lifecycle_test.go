package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeGracefullyDrainsInFlightRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	httpServer := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		response.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(response, "done")
	})}
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(ctx, httpServer, listener, 2*time.Second)
	}()

	responseDone := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr != nil {
			responseDone <- requestErr
			return
		}
		defer response.Body.Close()
		body, requestErr := io.ReadAll(response.Body)
		if requestErr == nil && string(body) != "done" {
			requestErr = &bodyMismatch{got: string(body)}
		}
		responseDone <- requestErr
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach handler")
	}
	cancel()
	select {
	case err := <-serveDone:
		t.Fatalf("Serve() returned before the request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)

	if err := <-responseDone; err != nil {
		t.Fatalf("in-flight response failed: %v", err)
	}
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve() error = %v, want nil", err)
	}
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("listener still accepted connections after shutdown")
	}
}

type bodyMismatch struct {
	got string
}

func (e *bodyMismatch) Error() string {
	return "response body = " + strings.TrimSpace(e.got) + ", want done"
}
