package link

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// TestLiveTailcatPairingReusesClient is intentionally excluded from the
// deterministic test gate. It contacts Tailcat's public DERP discovery and
// relay infrastructure with fresh in-memory keys.
func TestLiveTailcatPairingReusesClient(t *testing.T) {
	if os.Getenv("NEW_DELEGATE_LIVE_TAILCAT") != "1" {
		t.Skip("set NEW_DELEGATE_LIVE_TAILCAT=1 to run the live Tailcat smoke test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go serveLiveEchoBackend(ctx, backend)

	pairingReader, pairingWriter := io.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		networkDialer := new(net.Dialer)
		err := ServeTailcat(ctx, 18080, func(dialContext context.Context) (net.Conn, error) {
			return networkDialer.DialContext(dialContext, "tcp", backend.Addr().String())
		}, pairingWriter, 5*time.Second)
		_ = pairingWriter.CloseWithError(err)
		serverDone <- err
	}()

	pairing, err := ReadPairing(ctx, pairingReader)
	if err != nil {
		t.Fatalf("read live pairing: %v", err)
	}
	dialer, err := NewTailcatDialer(pairing)
	if err != nil {
		t.Fatalf("create live dialer: %v", err)
	}
	defer func() {
		drainContext, stopDrain := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopDrain()
		_ = dialer.Close(drainContext)
	}()

	for _, payload := range [][]byte{
		[]byte("first live Tailcat stream"),
		[]byte("second live Tailcat stream"),
	} {
		connection, err := dialer.Dial(ctx)
		if err != nil {
			t.Fatalf("dial live stream: %v", err)
		}
		if err := connection.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
			_ = connection.Close()
			t.Fatal(err)
		}
		if _, err := connection.Write(payload); err != nil {
			_ = connection.Close()
			t.Fatalf("write live stream: %v", err)
		}
		if closeWriter, ok := connection.(interface{ CloseWrite() error }); ok {
			if err := closeWriter.CloseWrite(); err != nil {
				_ = connection.Close()
				t.Fatalf("half-close live stream: %v", err)
			}
		}
		response := make([]byte, len(payload))
		if _, err := io.ReadFull(connection, response); err != nil {
			_ = connection.Close()
			t.Fatalf("read live stream: %v", err)
		}
		if string(response) != string(payload) {
			_ = connection.Close()
			t.Fatalf("live response = %q, want %q", response, payload)
		}
		if err := connection.Close(); err != nil {
			t.Fatalf("close live stream: %v", err)
		}
	}

	drainContext, stopDrain := context.WithTimeout(context.Background(), 5*time.Second)
	err = dialer.Close(drainContext)
	stopDrain()
	if err != nil {
		t.Fatalf("close live Tailcat client: %v", err)
	}
	cancel()
	select {
	case err := <-serverDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("stop live Tailcat server: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("live Tailcat server did not stop")
	}
}

func serveLiveEchoBackend(ctx context.Context, listener net.Listener) {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer connection.Close()
			_, _ = io.Copy(connection, connection)
		}()
	}
}
