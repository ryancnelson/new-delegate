package link

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestServeBridgesBytes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	remoteConnections := make(chan net.Conn, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, listener, func(context.Context) (net.Conn, error) {
			bridge, remote := net.Pipe()
			remoteConnections <- remote
			return bridge, nil
		}, time.Second)
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	remote := <-remoteConnections
	defer remote.Close()
	backendDone := make(chan error, 1)
	go func() {
		request := make([]byte, 4)
		if _, err := io.ReadFull(remote, request); err != nil {
			backendDone <- err
			return
		}
		if string(request) != "ping" {
			backendDone <- errors.New("unexpected request")
			return
		}
		_, err := remote.Write([]byte("pong"))
		backendDone <- err
	}()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 4)
	if _, err := io.ReadFull(client, response); err != nil {
		t.Fatal(err)
	}
	if string(response) != "pong" {
		t.Fatalf("response = %q, want %q", response, "pong")
	}
	if err := <-backendDone; err != nil {
		t.Fatal(err)
	}

	_ = client.Close()
	_ = remote.Close()
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve() = %v, want nil", err)
	}
}

func TestServeClosesClientWhenDialFails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, listener, func(context.Context) (net.Conn, error) {
			calls.Add(1)
			return nil, errors.New("unavailable")
		}, time.Second)
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 1)
	if _, err := client.Read(buffer); err == nil {
		t.Fatal("client read = nil, want closed connection")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("dial calls = %d, want 1", got)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve() = %v, want nil", err)
	}
}

func TestServeStopsDialingAfterCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var calls atomic.Int32
	if err := Serve(ctx, listener, func(context.Context) (net.Conn, error) {
		calls.Add(1)
		return nil, nil
	}, time.Second); err != nil {
		t.Fatalf("Serve() = %v, want nil", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("dial calls = %d, want 0", got)
	}
}
