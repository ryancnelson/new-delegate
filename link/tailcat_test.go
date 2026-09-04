package link

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPairingRoundTrip(t *testing.T) {
	t.Parallel()

	pairing := Pairing{RemotePort: 8080, Address: "tcExampleAddress"}
	encoded := pairing.Encode()
	if strings.Contains(encoded, "\n") {
		t.Fatalf("Encode() contains newline: %q", encoded)
	}
	decoded, err := ParsePairing(encoded)
	if err != nil {
		t.Fatalf("ParsePairing() = %v", err)
	}
	if decoded != pairing {
		t.Fatalf("ParsePairing() = %#v, want %#v", decoded, pairing)
	}
}

func TestPairingRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	for _, encoded := range []string{
		"", "tcExampleAddress", "ndlink1:0:tcExampleAddress",
		"ndlink2:8080:tcExampleAddress", "ndlink1:8080:not-tailcat",
		"ndlink1:8080:tcExample Address",
	} {
		if _, err := ParsePairing(encoded); err == nil {
			t.Fatalf("ParsePairing(%q) = nil, want error", encoded)
		}
	}
}

func TestTailcatArgumentsRequireLoopbackTargets(t *testing.T) {
	t.Parallel()

	if _, _, err := TailcatServeArgs("192.0.2.20:8080"); err == nil {
		t.Fatal("TailcatServeArgs() = nil, want non-loopback error")
	}
	pairing := Pairing{RemotePort: 8080, Address: "tcExampleAddress"}
	if _, err := TailcatForwardArgs(pairing, "0.0.0.0:18080"); err == nil {
		t.Fatal("TailcatForwardArgs() = nil, want non-loopback error")
	}
}

func TestRunRightPrintsOnePairingAndRedactsTailcatOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}

	binary := writeTailcatScript(t, `
printf 'server address tcExampleAddress\n'
while :; do sleep 1; done
`)
	ctx, cancel := context.WithCancel(context.Background())
	pairing := newSignalBuffer()
	diagnostics := newSignalBuffer()
	done := make(chan error, 1)
	joined := false
	go func() {
		done <- RunRight(ctx, binary, "127.0.0.1:8080", pairing, diagnostics)
	}()
	t.Cleanup(func() {
		if joined {
			return
		}
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("RunRight did not stop during cleanup")
		}
	})

	select {
	case <-pairing.changed:
	case <-time.After(time.Second):
		t.Fatal("RunRight did not emit a pairing")
	}
	if got, want := pairing.String(), "ndlink1:8080:tcExampleAddress\n"; got != want {
		t.Fatalf("pairing = %q, want %q", got, want)
	}

	cancel()
	select {
	case err := <-done:
		joined = true
		if err != nil {
			t.Fatalf("RunRight() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunRight did not stop after cancellation")
	}
	if strings.Contains(diagnostics.String(), "tcExampleAddress") {
		t.Fatalf("diagnostics leaked pairing address: %q", diagnostics.String())
	}
}

func TestRunLeftPassesPairingOnlyToTailcatChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}

	argsFile := filepath.Join(t.TempDir(), "args")
	binary := writeTailcatScript(t, `
printf '%s\n' "$@" > "$TAILCAT_ARGS_FILE"
while :; do sleep 1; done
`)
	t.Setenv("TAILCAT_ARGS_FILE", argsFile)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- RunLeft(ctx, binary, strings.NewReader("ndlink1:8080:tcExampleAddress\n"), "127.0.0.1:18080", io.Discard)
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(argsFile)
		if err == nil {
			if got, want := string(contents), "forward\ntcExampleAddress\n18080:8080\n"; got != want {
				t.Fatalf("tailcat arguments = %q, want %q", got, want)
			}
			cancel()
			if err := <-done; err != nil {
				t.Fatalf("RunLeft() = %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("tailcat child did not receive arguments")
}

func writeTailcatScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tailcat")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

type signalBuffer struct {
	mu      sync.Mutex
	value   strings.Builder
	changed chan struct{}
}

func newSignalBuffer() *signalBuffer {
	return &signalBuffer{changed: make(chan struct{}, 1)}
}

func (b *signalBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	written, err := b.value.Write(value)
	b.mu.Unlock()
	select {
	case b.changed <- struct{}{}:
	default:
	}
	return written, err
}

func (b *signalBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.value.String()
}
