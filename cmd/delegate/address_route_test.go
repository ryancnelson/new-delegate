package main

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/endpoint"
)

func TestServeAddressRouteRejectsUnsupportedCombinationBeforeNetwork(t *testing.T) {
	listenCalls := 0
	dialCalls := 0
	err := serveAddressRoute(context.Background(), endpoint.Route{
		Ingress: endpoint.Address{Kind: endpoint.TailcatListen, Port: 8080},
		Egress:  endpoint.Address{Kind: endpoint.TCPConnect, Host: "127.0.0.1", Port: 8080},
	}, addressRouteRuntime{
		listen: func(string, string) (net.Listener, error) {
			listenCalls++
			return nil, errors.New("must not listen")
		},
		dial: func(context.Context, string, string) (net.Conn, error) {
			dialCalls++
			return nil, errors.New("must not dial")
		},
		drain: time.Second,
	})

	if err == nil || listenCalls != 0 || dialCalls != 0 {
		t.Fatalf("serveAddressRoute() error = %v, listen calls=%d, dial calls=%d", err, listenCalls, dialCalls)
	}
}

func TestServeAddressRouteBridgesRepeatedTCPClients(t *testing.T) {
	frontend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = frontend.Close()
		t.Fatal(err)
	}
	defer backend.Close()

	backendDone := make(chan struct{})
	go func() {
		defer close(backendDone)
		for accepted := 0; accepted < 2; accepted++ {
			connection, acceptErr := backend.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer connection.Close()
				_, _ = io.Copy(connection, connection)
			}()
		}
	}()

	route := endpoint.Route{
		Ingress: endpoint.Address{Kind: endpoint.TCPListen, Host: "127.0.0.1", Port: 18080},
		Egress:  endpoint.Address{Kind: endpoint.TCPConnect, Host: "backend.internal", Port: 8080},
	}
	var listenCalls, dialCalls atomic.Int32
	networkDialer := new(net.Dialer)
	runtime := addressRouteRuntime{
		listen: func(network, address string) (net.Listener, error) {
			listenCalls.Add(1)
			if network != "tcp" || address != "127.0.0.1:18080" {
				return nil, errors.New("unexpected listen address")
			}
			return frontend, nil
		},
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialCalls.Add(1)
			if network != "tcp" || address != "backend.internal:8080" {
				t.Errorf("dial(%q, %q), want tcp, backend.internal:8080", network, address)
			}
			return networkDialer.DialContext(ctx, "tcp", backend.Addr().String())
		},
		drain: time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- serveAddressRoute(ctx, route, runtime) }()

	for _, payload := range []string{"first stream", "second stream"} {
		connection, err := net.Dial("tcp", frontend.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.Write([]byte(payload)); err != nil {
			t.Fatal(err)
		}
		response := make([]byte, len(payload))
		if _, err := io.ReadFull(connection, response); err != nil {
			t.Fatal(err)
		}
		if string(response) != payload {
			t.Fatalf("response = %q, want %q", response, payload)
		}
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("serveAddressRoute() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveAddressRoute did not stop after cancellation")
	}
	<-backendDone
	if listenCalls.Load() != 1 || dialCalls.Load() != 2 {
		t.Fatalf("listen calls = %d, dial calls = %d; want 1, 2", listenCalls.Load(), dialCalls.Load())
	}
}
