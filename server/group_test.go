package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeAllRunsAndStopsEveryListener(t *testing.T) {
	listeners := make([]net.Listener, 2)
	servers := make([]*http.Server, 2)
	for i := range listeners {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = listener
		servers[i] = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ready")
		})}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- ServeAll(ctx, servers, listeners, time.Second) }()

	client := &http.Client{Timeout: time.Second}
	for _, listener := range listeners {
		response, err := client.Get("http://" + listener.Addr().String())
		if err != nil {
			t.Fatalf("GET %s: %v", listener.Addr(), err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", listener.Addr(), response.StatusCode)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeAll() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeAll() did not stop every listener")
	}
	for _, listener := range listeners {
		connection, err := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			t.Fatalf("listener %s still accepts connections", listener.Addr())
		}
	}
}

func TestServeAllRejectsMismatchedInputs(t *testing.T) {
	err := ServeAll(context.Background(), []*http.Server{{}}, nil, time.Second)
	if err == nil {
		t.Fatal("ServeAll() error = nil, want mismatched input error")
	}
}
