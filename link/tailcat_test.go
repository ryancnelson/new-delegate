package link

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tailscale/tailcat"
)

const testTailcatAddress = "tcomFwWCCcjS5nKNqAod034nWoJZW0LZqDhhC8U_dKdnDRYQ8uNGFpGQEu"

func TestPairingRoundTrip(t *testing.T) {
	t.Parallel()

	pairing := Pairing{RemotePort: 8080, Address: testTailcatAddress}
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
		"ndlink1:8080:tcExample Address", "ndlink1:8080:tcp",
		" ndlink1:8080:" + testTailcatAddress,
		"ndlink1:8080:" + testTailcatAddress + "\n",
	} {
		if _, err := ParsePairing(encoded); err == nil {
			t.Fatalf("ParsePairing(%q) = nil, want error", encoded)
		}
	}
}

func TestReadPairingCompletesAtOneNewlineWithoutEOF(t *testing.T) {
	t.Parallel()

	input := newLineThenBlockReadCloser("ndlink1:8080:" + testTailcatAddress + "\n")
	result := make(chan struct {
		pairing Pairing
		err     error
	}, 1)
	go func() {
		pairing, err := ReadPairing(context.Background(), input)
		result <- struct {
			pairing Pairing
			err     error
		}{pairing, err}
	}()

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("ReadPairing() error = %v", got.err)
		}
		if want := (Pairing{RemotePort: 8080, Address: testTailcatAddress}); got.pairing != want {
			t.Fatalf("ReadPairing() = %#v, want %#v", got.pairing, want)
		}
	case <-time.After(time.Second):
		t.Fatal("ReadPairing waited for EOF after a complete line")
	}
	select {
	case <-input.closed:
	default:
		t.Fatal("ReadPairing did not close consumed input")
	}
}

func TestReadPairingRejectsOversizedAndBufferedTrailingInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		strings.Repeat("x", maxPairingLine+1) + "\n",
		"ndlink1:8080:" + testTailcatAddress + "\ntrailing\n",
	} {
		if _, err := ReadPairing(context.Background(), io.NopCloser(strings.NewReader(input))); err == nil {
			t.Fatalf("ReadPairing(%q) = nil, want error", input)
		}
	}
}

func TestReadPairingCancellationClosesInput(t *testing.T) {
	t.Parallel()

	input := newBlockingReadCloser()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := ReadPairing(ctx, input)
		done <- err
	}()
	<-input.started
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ReadPairing() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReadPairing did not return after cancellation")
	}
	select {
	case <-input.closed:
	default:
		t.Fatal("ReadPairing did not close blocked input")
	}
}

func TestProductionTailcatServerUsesEphemeralKeysAndExactPort(t *testing.T) {
	t.Parallel()

	server := newProductionTailcatServer(8080, func(net.Conn) {})
	if !server.Key.IsZero() {
		t.Fatal("production server loaded a persistent node key")
	}
	if !server.PresharedKey.IsZero() {
		t.Fatal("production server loaded a persistent preshared key")
	}
	if server.DERPMapCache != nil {
		t.Fatal("production server configured persistent DERP map state")
	}
	if got := server.ServedTCPPorts; len(got) != 1 || got[0].First != 8080 || got[0].Last != 8080 {
		t.Fatalf("ServedTCPPorts = %#v, want only 8080", got)
	}
	if server.OnTCP(8081) != nil {
		t.Fatal("OnTCP accepted an unconfigured port")
	}
	if server.OnTCP(8080) == nil {
		t.Fatal("OnTCP rejected the configured port")
	}
}

func TestTailcatDialerReusesClientAcrossStreamsAndDrainsOnClose(t *testing.T) {
	t.Parallel()

	fake := &fakeTailcatClient{}
	factoryCalls := 0
	dialer, err := newTailcatDialer(Pairing{RemotePort: 8080, Address: testTailcatAddress}, func(address tailcat.Addr) tailcatClient {
		factoryCalls++
		if address != tailcat.Addr(testTailcatAddress) {
			t.Fatalf("client address = %q, want %q", address, testTailcatAddress)
		}
		return fake
	})
	if err != nil {
		t.Fatalf("newTailcatDialer() error = %v", err)
	}
	if factoryCalls != 1 {
		t.Fatalf("factory calls = %d, want 1", factoryCalls)
	}

	for range 2 {
		connection, err := dialer.Dial(context.Background())
		if err != nil {
			t.Fatalf("Dial() error = %v", err)
		}
		_ = connection.Close()
	}
	if got := fake.dialPorts(); len(got) != 2 || got[0] != 8080 || got[1] != 8080 {
		t.Fatalf("dial ports = %v, want [8080 8080]", got)
	}
	if err := dialer.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if fake.drainCount != 1 || fake.closeCount != 1 {
		t.Fatalf("drain/close calls = %d/%d, want 1/1", fake.drainCount, fake.closeCount)
	}
	if err := dialer.Close(context.Background()); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if fake.drainCount != 1 || fake.closeCount != 1 {
		t.Fatalf("second Close changed drain/close calls to %d/%d", fake.drainCount, fake.closeCount)
	}
	if _, err := dialer.Dial(context.Background()); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Dial() after Close error = %v, want net.ErrClosed", err)
	}
}

func TestTailcatDialerRejectsInvalidPairingBeforeCreatingClient(t *testing.T) {
	t.Parallel()

	called := false
	_, err := newTailcatDialer(Pairing{RemotePort: 8080, Address: "tcp"}, func(tailcat.Addr) tailcatClient {
		called = true
		return &fakeTailcatClient{}
	})
	if err == nil {
		t.Fatal("newTailcatDialer() = nil, want error")
	}
	if called {
		t.Fatal("invalid pairing created a Tailcat client")
	}
}

func TestServeTailcatEmitsPairingAndBridgesRepeatedStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := newFakeTailcatServer()
	pairingOutput := newSignalBuffer()
	remoteConnections := make(chan net.Conn, 2)
	done := make(chan error, 1)
	go func() {
		done <- serveTailcat(ctx, 8080, func(context.Context) (net.Conn, error) {
			bridge, remote := net.Pipe()
			remoteConnections <- remote
			return bridge, nil
		}, pairingOutput, time.Second, func(port uint16, handler func(net.Conn)) tailcatServer {
			if port != 8080 {
				t.Errorf("factory port = %d, want 8080", port)
			}
			fake.handler = handler
			return fake
		})
	}()

	select {
	case <-pairingOutput.changed:
	case <-time.After(time.Second):
		t.Fatal("ServeTailcat did not emit a pairing")
	}
	if got, want := pairingOutput.String(), "ndlink1:8080:"+testTailcatAddress+"\n"; got != want {
		t.Fatalf("pairing output = %q, want %q", got, want)
	}

	for i := byte(1); i <= 2; i++ {
		serverConnection, clientConnection := net.Pipe()
		go fake.handler(serverConnection)
		remote := <-remoteConnections
		if _, err := clientConnection.Write([]byte{i}); err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(remote, buffer); err != nil {
			t.Fatal(err)
		}
		if buffer[0] != i {
			t.Fatalf("relayed byte = %d, want %d", buffer[0], i)
		}
		_ = clientConnection.Close()
		_ = remote.Close()
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeTailcat() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeTailcat did not stop after cancellation")
	}
	if fake.drainCount != 1 || fake.closeCount != 1 {
		t.Fatalf("drain/close calls = %d/%d, want 1/1", fake.drainCount, fake.closeCount)
	}
}

func TestServeTailcatOutputFailureClosesStartedServer(t *testing.T) {
	t.Parallel()

	fake := newFakeTailcatServer()
	err := serveTailcat(context.Background(), 8080, func(context.Context) (net.Conn, error) {
		return nil, errors.New("unused")
	}, errorWriter{}, time.Second, func(_ uint16, handler func(net.Conn)) tailcatServer {
		fake.handler = handler
		return fake
	})
	if err == nil || !strings.Contains(err.Error(), "write pairing") {
		t.Fatalf("ServeTailcat() error = %v, want write pairing error", err)
	}
	if fake.closeCount != 1 {
		t.Fatalf("server close calls = %d, want 1", fake.closeCount)
	}
}

func TestServeTailcatCancellationDuringStartupReturnsAndReapsServer(t *testing.T) {
	t.Parallel()

	fake := newFakeTailcatServer()
	fake.startRelease = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveTailcat(ctx, 8080, func(context.Context) (net.Conn, error) {
			return nil, errors.New("unused")
		}, io.Discard, time.Second, func(_ uint16, handler func(net.Conn)) tailcatServer {
			fake.handler = handler
			return fake
		})
	}()
	<-fake.startCalled
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeTailcat() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeTailcat did not return during blocked startup")
	}
	close(fake.startRelease)
	select {
	case <-fake.closed:
	case <-time.After(time.Second):
		t.Fatal("server was not closed after canceled startup completed")
	}
}

func TestServeTailcatStartupFailureClosesServer(t *testing.T) {
	t.Parallel()

	fake := newFakeTailcatServer()
	fake.startErr = errors.New("cannot start")
	err := serveTailcat(context.Background(), 8080, func(context.Context) (net.Conn, error) {
		return nil, errors.New("unused")
	}, io.Discard, time.Second, func(_ uint16, handler func(net.Conn)) tailcatServer {
		fake.handler = handler
		return fake
	})
	if err == nil || !strings.Contains(err.Error(), "cannot start") {
		t.Fatalf("ServeTailcat() error = %v, want startup error", err)
	}
	if fake.closeCount != 1 {
		t.Fatalf("server close calls = %d, want 1", fake.closeCount)
	}
}

type fakeTailcatClient struct {
	mu         sync.Mutex
	ports      []uint16
	drainCount int
	closeCount int
}

func (c *fakeTailcatClient) DialTCPPort(_ context.Context, port uint16) (net.Conn, error) {
	c.mu.Lock()
	c.ports = append(c.ports, port)
	c.mu.Unlock()
	connection, peer := net.Pipe()
	_ = peer.Close()
	return connection, nil
}

func (c *fakeTailcatClient) DrainTCP(context.Context) error {
	c.drainCount++
	return nil
}

func (c *fakeTailcatClient) Close() error {
	c.closeCount++
	return nil
}

func (c *fakeTailcatClient) dialPorts() []uint16 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]uint16(nil), c.ports...)
}

type fakeTailcatServer struct {
	startCalled  chan struct{}
	startRelease chan struct{}
	closed       chan struct{}
	closeOnce    sync.Once
	handler      func(net.Conn)
	startErr     error
	drainCount   int
	closeCount   int
}

func newFakeTailcatServer() *fakeTailcatServer {
	return &fakeTailcatServer{
		startCalled: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (s *fakeTailcatServer) Start() error {
	close(s.startCalled)
	if s.startRelease != nil {
		<-s.startRelease
	}
	return s.startErr
}

func (s *fakeTailcatServer) TailcatAddr() tailcat.Addr {
	return tailcat.Addr(testTailcatAddress)
}

func (s *fakeTailcatServer) DrainTCP(context.Context) error {
	s.drainCount++
	return nil
}

func (s *fakeTailcatServer) Close() error {
	s.closeCount++
	s.closeOnce.Do(func() { close(s.closed) })
	return nil
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type lineThenBlockReadCloser struct {
	line   []byte
	closed chan struct{}
	once   sync.Once
}

func newLineThenBlockReadCloser(line string) *lineThenBlockReadCloser {
	return &lineThenBlockReadCloser{line: []byte(line), closed: make(chan struct{})}
}

func (r *lineThenBlockReadCloser) Read(buffer []byte) (int, error) {
	if len(r.line) > 0 {
		count := copy(buffer, r.line)
		r.line = r.line[count:]
		return count, nil
	}
	<-r.closed
	return 0, io.EOF
}

func (r *lineThenBlockReadCloser) Close() error {
	r.once.Do(func() { close(r.closed) })
	return nil
}

type blockingReadCloser struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{started: make(chan struct{}), closed: make(chan struct{})}
}

func (r *blockingReadCloser) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.closed
	return 0, io.EOF
}

func (r *blockingReadCloser) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
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

func FuzzParsePairing(f *testing.F) {
	for _, seed := range []string{
		"",
		"tcp",
		"ndlink1:8080:" + testTailcatAddress,
		"ndlink1:0:" + testTailcatAddress,
		" ndlink1:8080:" + testTailcatAddress,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		pairing, err := ParsePairing(input)
		if err != nil {
			return
		}
		if pairing.RemotePort == 0 || !validTailcatAddress(pairing.Address) {
			t.Fatalf("ParsePairing(%q) returned invalid pairing %#v", input, pairing)
		}
	})
}
