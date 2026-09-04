package connector

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/operation"
)

func TestFTPUsesEPSVAndControlPeerForIPv6Data(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	peerIP := net.ParseIP("2001:db8::7")
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: peerIP, Port: 21},
		},
		data: dataClient,
	}
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := ftpLoginScript(reader, control); err != nil {
			return err
		}
		if err := expectFTPCommand(reader, "EPSV"); err != nil {
			return err
		}
		if err := writeFTPLine(control, "229 Entering Extended Passive Mode (|||4242|)"); err != nil {
			return err
		}
		return ftpRetrieveScript(reader, control, dataServer, "ipv6 payload")
	})

	result, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://[2001:db8::7]:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	payload, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(payload), "ipv6 payload"; got != want {
		t.Fatalf("payload = %q, want %q", got, want)
	}
	if got, want := dialer.addresses(), []string{"[2001:db8::7]:21", "[2001:db8::7]:4242"}; !equalStrings(got, want) {
		t.Fatalf("dial addresses = %q, want %q", got, want)
	}
	if err := <-scriptDone; err != nil {
		t.Fatal(err)
	}
}

func TestFTPPASVFallbackIgnoresAdvertisedHost(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		data: dataClient,
	}
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := ftpLoginScript(reader, control); err != nil {
			return err
		}
		if err := expectFTPCommand(reader, "EPSV"); err != nil {
			return err
		}
		if err := writeFTPLine(control, "500 EPSV unsupported"); err != nil {
			return err
		}
		if err := expectFTPCommand(reader, "PASV"); err != nil {
			return err
		}
		if err := writeFTPLine(control, "227 Entering Passive Mode (203,0,113,77,16,146)"); err != nil {
			return err
		}
		return ftpRetrieveScript(reader, control, dataServer, "fallback payload")
	})

	result, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if got, want := dialer.addresses(), []string{"127.0.0.1:21", "127.0.0.1:4242"}; !equalStrings(got, want) {
		t.Fatalf("dial addresses = %q, want %q", got, want)
	}
	if err := <-scriptDone; err != nil {
		t.Fatal(err)
	}
}

func TestFTPPASVFallbackRejectsInvalidAddressFieldsBeforeDial(t *testing.T) {
	for _, reply := range []string{
		"227 Entering Passive Mode (127,0,0,x,16,146)",
		"227 Entering Passive Mode (127,0,0,999,16,146)",
		"227 Entering Passive Mode (127,0,0,1,0,0)",
		"227 Entering Passive Mode (127,0,0,1,256,0)",
	} {
		t.Run(reply, func(t *testing.T) {
			controlClient, controlServer := net.Pipe()
			dialer := &ftpScriptDialer{
				control: ftpRemoteAddrConn{
					Conn: controlClient,
					peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
				},
				dataErr: errors.New("unexpected data dial"),
			}
			scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
				if err := ftpLoginScript(reader, control); err != nil {
					return err
				}
				command, err := readFTPCommand(reader)
				if err != nil {
					return err
				}
				if command == "EPSV" {
					if err := writeFTPLine(control, "500 EPSV unsupported"); err != nil {
						return err
					}
					command, err = readFTPCommand(reader)
					if err != nil {
						return err
					}
				}
				if command != "PASV" {
					return fmt.Errorf("command = %q, want PASV", command)
				}
				return writeFTPLine(control, reply)
			})

			_, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
				Method:   "GET",
				Resource: "ftp://127.0.0.1:21/file.txt",
			})
			if err == nil {
				t.Fatal("Fetch() error = nil, want invalid passive response")
			}
			if got, want := len(dialer.addresses()), 1; got != want {
				t.Fatalf("dial count = %d, want %d; addresses = %q", got, want, dialer.addresses())
			}
			if scriptErr := <-scriptDone; scriptErr != nil {
				t.Fatal(scriptErr)
			}
		})
	}
}

func TestFTPDoesNotFallbackAfterNonCapabilityEPSVFailure(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		dataErr: errors.New("unexpected data dial"),
	}
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := ftpLoginScript(reader, control); err != nil {
			return err
		}
		if err := expectFTPCommand(reader, "EPSV"); err != nil {
			return err
		}
		return writeFTPLine(control, "530 Not logged in")
	})

	_, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err == nil || !strings.Contains(err.Error(), "EPSV rejected with 530") {
		t.Fatalf("Fetch() error = %v, want EPSV rejection", err)
	}
	if got, want := len(dialer.addresses()), 1; got != want {
		t.Fatalf("dial count = %d, want %d; addresses = %q", got, want, dialer.addresses())
	}
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestParseEPSVPort(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want int
	}{
		{name: "standard delimiter", body: "229 Entering Extended Passive Mode (|||4242|)", want: 4242},
		{name: "alternate delimiter", body: "229 Entering Extended Passive Mode (!!!443!)", want: 443},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseEPSVPort(test.body)
			if err != nil || got != test.want {
				t.Fatalf("parseEPSVPort(%q) = %d, %v; want %d", test.body, got, err, test.want)
			}
		})
	}

	for _, body := range []string{
		"229 no tuple",
		"229 (||4242|)",
		"229 (|||not-a-port|)",
		"229 (|||0|)",
		"229 (|||65536|)",
	} {
		t.Run(body, func(t *testing.T) {
			if port, err := parseEPSVPort(body); err == nil {
				t.Fatalf("parseEPSVPort(%q) = %d, nil; want error", body, port)
			}
		})
	}
}

func TestParsePASVPort(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want int
	}{
		{name: "ordinary", body: "227 Entering Passive Mode (127,0,0,1,16,146)", want: 4242},
		{name: "whitespace", body: "227 Passive ( 10, 0, 0, 1, 1, 187 )", want: 443},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePASVPort(test.body)
			if err != nil || got != test.want {
				t.Fatalf("parsePASVPort(%q) = %d, %v; want %d", test.body, got, err, test.want)
			}
		})
	}

	for _, body := range []string{
		"227 no tuple",
		"227 (127,0,0,1,1)",
		"227 (127,0,0,host,1,187)",
		"227 (127,0,0,256,1,187)",
		"227 (127,0,0,1,0,0)",
	} {
		t.Run(body, func(t *testing.T) {
			if port, err := parsePASVPort(body); err == nil {
				t.Fatalf("parsePASVPort(%q) = %d, nil; want error", body, port)
			}
		})
	}
}

func FuzzParseEPSVPort(f *testing.F) {
	for _, seed := range []string{
		"229 Entering Extended Passive Mode (|||4242|)",
		"229 (!!!443!)",
		"229 (|||0|)",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		port, err := parseEPSVPort(body)
		if err == nil && (port < 1 || port > 65535) {
			t.Fatalf("parseEPSVPort(%q) returned invalid port %d", body, port)
		}
	})
}

func FuzzParsePASVPort(f *testing.F) {
	for _, seed := range []string{
		"227 Entering Passive Mode (127,0,0,1,16,146)",
		"227 (127,0,0,1,0,0)",
		"227 (127,0,0,x,16,146)",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		port, err := parsePASVPort(body)
		if err == nil && (port < 1 || port > 65535) {
			t.Fatalf("parsePASVPort(%q) returned invalid port %d", body, port)
		}
	})
}

func TestFTPReadsMultilineGreeting(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		data: dataClient,
	}
	scriptDone := make(chan error, 1)
	go func() {
		defer controlServer.Close()
		if _, err := io.WriteString(controlServer, "220-Welcome\r\n220 Ready\r\n"); err != nil {
			scriptDone <- err
			return
		}
		reader := bufio.NewReader(controlServer)
		if err := ftpLoginScript(reader, controlServer); err != nil {
			scriptDone <- err
			return
		}
		if err := expectFTPCommand(reader, "EPSV"); err != nil {
			scriptDone <- err
			return
		}
		if err := writeFTPLine(controlServer, "229 Entering Extended Passive Mode (|||4242|)"); err != nil {
			scriptDone <- err
			return
		}
		scriptDone <- ftpRetrieveScript(reader, controlServer, dataServer, "multiline greeting")
	}()

	result, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if err := <-scriptDone; err != nil {
		t.Fatal(err)
	}
}

func TestFTPRejectsOversizedMultilineGreetingBeforeCommand(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	tracked := &ftpCloseTrackingConn{
		Conn: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		closed: make(chan struct{}),
	}
	dialer := &ftpScriptDialer{control: tracked}
	connector := NewFTP(dialer)
	connector.maxReplyBytes = 64
	scriptDone := make(chan error, 1)
	go func() {
		defer controlServer.Close()
		_, err := io.WriteString(controlServer, "220-"+strings.Repeat("x", 80)+"\r\n220 Ready\r\n")
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.ErrClosedPipe) {
			scriptDone <- err
			return
		}
		scriptDone <- nil
	}()

	_, err := connector.Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err == nil || !strings.Contains(err.Error(), "reply exceeds") {
		t.Fatalf("Fetch() error = %v, want bounded reply error", err)
	}
	select {
	case <-tracked.closed:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("control connection was not closed after oversized greeting")
	}
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestFTPStalledGreetingUsesControlTimeout(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	defer controlServer.Close()
	tracked := &ftpCloseTrackingConn{
		Conn: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		closed: make(chan struct{}),
	}
	dialer := &ftpScriptDialer{control: tracked}
	connector := NewFTP(dialer)
	connector.controlTimeout = 25 * time.Millisecond

	started := time.Now()
	_, err := connector.Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Fetch() error = %v, want control timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("stalled greeting returned after %s, want bounded return", elapsed)
	}
	select {
	case <-tracked.closed:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("control connection was not closed after greeting timeout")
	}
}

func TestFTPStalledGreetingHonorsCancellation(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	defer controlServer.Close()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
	}
	connector := NewFTP(dialer)
	connector.controlTimeout = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := connector.Fetch(ctx, operation.Fetch{
			Method:   "GET",
			Resource: "ftp://127.0.0.1:21/file.txt",
		})
		result <- err
	}()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Fetch() error = %v, want context cancellation", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Fetch did not return after cancellation")
	}
}

func TestFTPDialUsesBoundedContext(t *testing.T) {
	connector := NewFTP(ftpDialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}))
	connector.dialTimeout = 25 * time.Millisecond
	started := time.Now()
	_, err := connector.Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Fetch() error = %v, want dial deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("dial returned after %s, want bounded return", elapsed)
	}
}

func TestReadFTPReplyRejectsMalformedMultiline(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		max   int
	}{
		{name: "unterminated", input: "220-first\r\nmore text\r\n", max: 256},
		{name: "wrong terminal code", input: "220-first\r\n221 done\r\n", max: 256},
		{name: "oversized total", input: "220-first\r\n" + strings.Repeat("x", 80) + "\r\n", max: 64},
		{name: "bad separator", input: "220!bad\r\n", max: 256},
	} {
		t.Run(test.name, func(t *testing.T) {
			if reply, err := newFTPReplyReader(strings.NewReader(test.input), test.max).ReadReply(); err == nil {
				t.Fatalf("ReadReply(%q) = %#v, nil; want error", test.input, reply)
			}
		})
	}
}

func FuzzReadFTPReply(f *testing.F) {
	for _, seed := range []string{
		"220 ready\r\n",
		"220-first\r\n220 ready\r\n",
		"bad\r\n",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		reader := newFTPReplyReader(strings.NewReader(input), 256)
		reply, err := reader.ReadReply()
		if err == nil && (reply.Code < 100 || reply.Code > 599) {
			t.Fatalf("ReadReply(%q) returned invalid code %d", input, reply.Code)
		}
	})
}

type ftpRemoteAddrConn struct {
	net.Conn
	peer net.Addr
}

func (c ftpRemoteAddrConn) RemoteAddr() net.Addr { return c.peer }

type ftpCloseTrackingConn struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
}

func (c *ftpCloseTrackingConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type ftpScriptDialer struct {
	mu      sync.Mutex
	control net.Conn
	data    net.Conn
	dataErr error
	dials   []string
}

type ftpDialerFunc func(context.Context, string, string) (net.Conn, error)

func (f ftpDialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func (d *ftpScriptDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dials = append(d.dials, address)
	if len(d.dials) == 1 {
		return d.control, nil
	}
	if d.dataErr != nil {
		return nil, d.dataErr
	}
	if d.data == nil {
		return nil, errors.New("unexpected data dial")
	}
	return d.data, nil
}

func (d *ftpScriptDialer) addresses() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.dials...)
}

func runFTPControlScript(control net.Conn, script func(*bufio.Reader, net.Conn) error) <-chan error {
	done := make(chan error, 1)
	go func() {
		defer control.Close()
		if err := writeFTPLine(control, "220 welcome"); err != nil {
			done <- err
			return
		}
		done <- script(bufio.NewReader(control), control)
	}()
	return done
}

func ftpLoginScript(reader *bufio.Reader, control net.Conn) error {
	if err := expectFTPCommand(reader, "USER"); err != nil {
		return err
	}
	if err := writeFTPLine(control, "230 User logged in"); err != nil {
		return err
	}
	if err := expectFTPCommand(reader, "TYPE"); err != nil {
		return err
	}
	return writeFTPLine(control, "200 Type set to I")
}

func ftpRetrieveScript(reader *bufio.Reader, control net.Conn, data net.Conn, payload string) error {
	defer data.Close()
	if err := expectFTPCommand(reader, "RETR"); err != nil {
		return err
	}
	if err := writeFTPLine(control, "150 Opening data connection"); err != nil {
		return err
	}
	if _, err := io.WriteString(data, payload); err != nil {
		return err
	}
	if err := data.Close(); err != nil {
		return err
	}
	return writeFTPLine(control, "226 Transfer complete")
}

func expectFTPCommand(reader *bufio.Reader, want string) error {
	got, err := readFTPCommand(reader)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("command = %q, want %q", got, want)
	}
	return nil
}

func readFTPCommand(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return "", errors.New("empty FTP command")
	}
	return strings.ToUpper(fields[0]), nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
