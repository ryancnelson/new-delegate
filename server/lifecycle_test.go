package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/policy"
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

func TestServeCancelsPendingCONNECTDial(t *testing.T) {
	dialStarted := make(chan struct{})
	dialCanceled := make(chan struct{})
	relay := relayConnectorFunc(func(ctx context.Context, _ operation.Relay) (net.Conn, error) {
		close(dialStarted)
		<-ctx.Done()
		close(dialCanceled)
		return nil, ctx.Err()
	})
	listener, cancel, serveDone := startCONNECTServer(t, relay, time.Second)
	client := dialCONNECT(t, listener)
	defer client.Close()

	select {
	case <-dialStarted:
	case <-time.After(time.Second):
		t.Fatal("CONNECT dial did not start")
	}
	cancel()
	select {
	case <-dialCanceled:
	case <-time.After(time.Second):
		t.Fatal("server shutdown did not cancel the pending CONNECT dial")
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after the CONNECT dial was canceled")
	}
}

func TestServeLetsCONNECTFinishDuringDrain(t *testing.T) {
	backend, backendPeer := net.Pipe()
	relay := relayConnectorFunc(func(context.Context, operation.Relay) (net.Conn, error) {
		return backend, nil
	})
	listener, cancel, serveDone := startCONNECTServer(t, relay, time.Second)
	client := dialCONNECT(t, listener)
	readCONNECTResponse(t, client)

	cancel()
	select {
	case err := <-serveDone:
		t.Fatalf("Serve() returned while the CONNECT session was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	_ = client.Close()
	_ = backendPeer.Close()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after the CONNECT session drained")
	}
}

func TestServeDrainsCONNECTAfterListenerFailure(t *testing.T) {
	backend, backendPeer := net.Pipe()
	relay := relayConnectorFunc(func(context.Context, operation.Relay) (net.Conn, error) {
		return backend, nil
	})
	listener, _, serveDone := startCONNECTServer(t, relay, time.Second)
	client := dialCONNECT(t, listener)
	readCONNECTResponse(t, client)

	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serveDone:
		t.Fatalf("Serve() returned while the CONNECT session was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	_ = client.Close()
	_ = backendPeer.Close()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not drain CONNECT after listener failure")
	}
}

func TestServeForceClosesCONNECTAtDrainDeadline(t *testing.T) {
	backend, backendPeer := net.Pipe()
	defer backendPeer.Close()
	trackedBackend := &closeSignalConn{Conn: backend, closed: make(chan struct{})}
	relay := relayConnectorFunc(func(context.Context, operation.Relay) (net.Conn, error) {
		return trackedBackend, nil
	})
	listener, cancel, serveDone := startCONNECTServer(t, relay, 25*time.Millisecond)
	client := dialCONNECT(t, listener)
	defer client.Close()
	reader := readCONNECTResponse(t, client)

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not force-close CONNECT at the drain deadline")
	}
	select {
	case <-trackedBackend.closed:
	default:
		t.Fatal("CONNECT backend was still owned after Serve() returned")
	}
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := reader.ReadByte(); err == nil {
		t.Fatal("CONNECT client remained open after the drain deadline")
	} else if networkError, ok := err.(net.Error); ok && networkError.Timeout() {
		t.Fatal("CONNECT client timed out instead of being closed at the drain deadline")
	}
	_ = backendPeer.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := backendPeer.Read(make([]byte, 1)); err == nil {
		t.Fatal("CONNECT backend remained open after the drain deadline")
	}
}

func startCONNECTServer(t *testing.T, relay relayConnector, shutdownTimeout time.Duration) (net.Listener, context.CancelFunc, <-chan error) {
	t.Helper()
	configured := config.Config{
		Servers: []config.Server{{Name: "proxy", Protocol: "http", Listen: ":8080"}},
		Mounts: []mount.Mount{{
			Source: "connect://origin.example:443/", Target: "tcp://127.0.0.1:8443",
		}},
		Policies: []policy.Rule{{
			Effect: policy.Permit, Protocol: "http", Destination: "127.0.0.1",
			Source: "127.0.0.1", Method: http.MethodConnect,
		}},
	}
	handler := NewReloadableHTTPHandlerWithRoutesAndRelay(
		"proxy", func() config.Config { return configured }, lifecycleMountConnector{}, relay,
	)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, &http.Server{Handler: handler}, listener, shutdownTimeout)
	}()
	return listener, cancel, done
}

func dialCONNECT(t *testing.T, listener net.Listener) net.Conn {
	t.Helper()
	client, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(client, "CONNECT origin.example:443 HTTP/1.1\r\nHost: origin.example:443\r\n\r\n"); err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	return client
}

func readCONNECTResponse(t *testing.T, client net.Conn) *bufio.Reader {
	t.Helper()
	reader := bufio.NewReader(client)
	status, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(status, " 200 ") {
		t.Fatalf("CONNECT response = %q, error=%v", status, err)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			return reader
		}
	}
}

type relayConnectorFunc func(context.Context, operation.Relay) (net.Conn, error)

func (f relayConnectorFunc) Connect(ctx context.Context, relay operation.Relay) (net.Conn, error) {
	return f(ctx, relay)
}

type lifecycleMountConnector struct{}

func (lifecycleMountConnector) FetchForMount(context.Context, mount.Mount, operation.Fetch) (operation.Result, error) {
	return operation.Result{}, operation.ErrUnsupported
}

type closeSignalConn struct {
	net.Conn
	once   sync.Once
	closed chan struct{}
}

func (c *closeSignalConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type bodyMismatch struct {
	got string
}

func (e *bodyMismatch) Error() string {
	return "response body = " + strings.TrimSpace(e.got) + ", want done"
}
