package link

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tailscale/tailcat"
	"tailscale.com/wgengine/filter"
)

const (
	maxPairingLine         = 4096
	tailcatListenerBacklog = 64
)

// ReadPairing owns and closes input after reading one bounded,
// newline-terminated pairing handoff. It does not wait for EOF. Bytes already
// buffered after the newline are rejected; later bytes cannot enter the
// process because the input has been closed.
func ReadPairing(ctx context.Context, input io.ReadCloser) (Pairing, error) {
	if input == nil {
		return Pairing{}, errors.New("pairing input is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	type result struct {
		pairing Pairing
		err     error
	}
	results := make(chan result, 1)
	go func() {
		pairing, err := readPairingLine(input)
		results <- result{pairing: pairing, err: err}
	}()

	select {
	case completed := <-results:
		_ = input.Close()
		if err := ctx.Err(); err != nil {
			return Pairing{}, err
		}
		return completed.pairing, completed.err
	case <-ctx.Done():
		_ = input.Close()
		return Pairing{}, ctx.Err()
	}
}

func readPairingLine(input io.Reader) (Pairing, error) {
	reader := bufio.NewReaderSize(input, maxPairingLine+2)
	line, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) || len(line) > maxPairingLine+1 {
		return Pairing{}, errors.New("link pairing exceeds maximum line length")
	}
	if err != nil {
		if errors.Is(err, io.EOF) {
			return Pairing{}, errors.New("link pairing must end with a newline")
		}
		return Pairing{}, fmt.Errorf("read link pairing: %w", err)
	}
	if reader.Buffered() != 0 {
		return Pairing{}, errors.New("link pairing contains trailing input")
	}
	return ParsePairing(strings.TrimSuffix(string(line), "\n"))
}

type tailcatClient interface {
	DialTCPPort(context.Context, uint16) (net.Conn, error)
	DrainTCP(context.Context) error
	Close() error
}

type tailcatClientFactory func(tailcat.Addr) tailcatClient

// TailcatDialer owns one Tailcat client and reuses its encrypted tunnel for
// every TCP stream opened through Dial.
type TailcatDialer struct {
	client    tailcatClient
	port      uint16
	closed    atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

// NewTailcatDialer validates pairing before allocating a lazy Tailcat client.
// No network access occurs until the first Dial.
func NewTailcatDialer(pairing Pairing) (*TailcatDialer, error) {
	return newTailcatDialer(pairing, func(address tailcat.Addr) tailcatClient {
		return &tailcat.Client{Server: address, Logf: discardTailcatLog}
	})
}

func newTailcatDialer(pairing Pairing, factory tailcatClientFactory) (*TailcatDialer, error) {
	if pairing.RemotePort == 0 || !validTailcatAddress(pairing.Address) {
		return nil, errors.New("invalid link pairing")
	}
	if factory == nil {
		return nil, errors.New("Tailcat client factory is required")
	}
	client := factory(tailcat.Addr(pairing.Address))
	if client == nil {
		return nil, errors.New("Tailcat client factory returned nil")
	}
	return &TailcatDialer{client: client, port: pairing.RemotePort}, nil
}

// Dial opens one stream through the reusable Tailcat client.
func (d *TailcatDialer) Dial(ctx context.Context) (net.Conn, error) {
	if d == nil || d.client == nil {
		return nil, errors.New("Tailcat dialer is not initialized")
	}
	if d.closed.Load() {
		return nil, net.ErrClosed
	}
	return d.client.DialTCPPort(ctx, d.port)
}

// Close stops new dials, gives Tailcat's userspace TCP stack the caller's
// context in which to flush final segments, then closes the client. It is
// idempotent.
func (d *TailcatDialer) Close(ctx context.Context) error {
	if d == nil || d.client == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d.closeOnce.Do(func() {
		d.closed.Store(true)
		d.closeErr = errors.Join(d.client.DrainTCP(ctx), d.client.Close())
	})
	return d.closeErr
}

type tailcatServer interface {
	Start() error
	TailcatAddr() tailcat.Addr
	DrainTCP(context.Context) error
	Close() error
}

type tailcatServerFactory func(port uint16, handler func(net.Conn)) tailcatServer

// ServeTailcat starts an ephemeral Tailcat ingress, emits its one-line pairing,
// and passes incoming streams through the shared owned/drained bridge.
func ServeTailcat(ctx context.Context, port uint16, dial Dialer, pairingOutput io.Writer, drain time.Duration) error {
	return serveTailcat(ctx, port, dial, pairingOutput, drain, func(port uint16, handler func(net.Conn)) tailcatServer {
		return newProductionTailcatServer(port, handler)
	})
}

func serveTailcat(ctx context.Context, port uint16, dial Dialer, pairingOutput io.Writer, drain time.Duration, factory tailcatServerFactory) (returnErr error) {
	if port == 0 {
		return errors.New("Tailcat listen port must be positive")
	}
	if dial == nil {
		return errors.New("link dialer is required")
	}
	if pairingOutput == nil {
		return errors.New("pairing output is required")
	}
	if drain <= 0 {
		return errors.New("link drain timeout must be positive")
	}
	if factory == nil {
		return errors.New("Tailcat server factory is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	listener := newTailcatListener(port)
	server := factory(port, listener.Handle)
	if server == nil {
		return errors.New("Tailcat server factory returned nil")
	}
	if err := startTailcatServer(ctx, server); err != nil {
		_ = listener.Close()
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, server.Close())
	}()
	defer listener.Close()

	address := server.TailcatAddr()
	if !validTailcatAddress(string(address)) {
		return errors.New("Tailcat server returned an invalid pairing address")
	}
	pairingLine := Pairing{RemotePort: port, Address: string(address)}.Encode() + "\n"
	written, err := io.WriteString(pairingOutput, pairingLine)
	if err != nil {
		return fmt.Errorf("write pairing: %w", err)
	}
	if written != len(pairingLine) {
		return fmt.Errorf("write pairing: %w", io.ErrShortWrite)
	}

	serveErr := Serve(ctx, listener, dial, drain)
	drainContext, cancelDrain := context.WithTimeout(context.Background(), drain)
	drainErr := server.DrainTCP(drainContext)
	cancelDrain()
	if errors.Is(drainErr, context.DeadlineExceeded) {
		drainErr = nil
	}
	return errors.Join(serveErr, drainErr)
}

func startTailcatServer(ctx context.Context, server tailcatServer) error {
	started := make(chan error, 1)
	go func() {
		started <- server.Start()
	}()
	select {
	case err := <-started:
		if err != nil {
			return errors.Join(fmt.Errorf("start Tailcat server: %w", err), server.Close())
		}
		return nil
	case <-ctx.Done():
		go func() {
			<-started
			_ = server.Close()
		}()
		return ctx.Err()
	}
}

func newProductionTailcatServer(port uint16, handler func(net.Conn)) *tailcat.Server {
	return &tailcat.Server{
		Logf: discardTailcatLog,
		ServedTCPPorts: []filter.PortRange{{
			First: port,
			Last:  port,
		}},
		OnTCP: func(requested uint16) func(net.Conn) {
			if requested != port {
				return nil
			}
			return handler
		},
	}
}

func discardTailcatLog(string, ...any) {}

type tailcatListener struct {
	port      uint16
	incoming  chan net.Conn
	closed    chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	closing   bool
}

func newTailcatListener(port uint16) *tailcatListener {
	return &tailcatListener{
		port:     port,
		incoming: make(chan net.Conn, tailcatListenerBacklog),
		closed:   make(chan struct{}),
	}
}

// Handle offers one Tailcat stream to Accept. A full local handoff queue fails
// closed instead of blocking Tailcat's userspace network stack indefinitely.
func (l *tailcatListener) Handle(connection net.Conn) {
	if connection == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closing {
		_ = connection.Close()
		return
	}
	select {
	case l.incoming <- connection:
	default:
		_ = connection.Close()
	}
}

func (l *tailcatListener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.closed:
			return nil, net.ErrClosed
		default:
		}
		select {
		case <-l.closed:
			return nil, net.ErrClosed
		case connection := <-l.incoming:
			l.mu.Lock()
			closing := l.closing
			l.mu.Unlock()
			if closing {
				_ = connection.Close()
				return nil, net.ErrClosed
			}
			return connection, nil
		}
	}
}

func (l *tailcatListener) Close() error {
	l.closeOnce.Do(func() {
		l.mu.Lock()
		l.closing = true
		close(l.closed)
		for {
			select {
			case connection := <-l.incoming:
				_ = connection.Close()
			default:
				l.mu.Unlock()
				return
			}
		}
	})
	return nil
}

func (l *tailcatListener) Addr() net.Addr {
	return tailcatListenerAddress{port: l.port}
}

type tailcatListenerAddress struct {
	port uint16
}

func (tailcatListenerAddress) Network() string { return "tailcat" }

func (a tailcatListenerAddress) String() string { return fmt.Sprintf("tailcat:%d", a.port) }
