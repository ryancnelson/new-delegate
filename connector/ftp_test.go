package connector

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/operation"
)

func TestFTPRejectsUnsupportedFetchMethodsBeforeDial(t *testing.T) {
	for _, method := range []string{"", "HEAD", "POST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			dials := 0
			connector := NewFTP(ftpDialerFunc(func(context.Context, string, string) (net.Conn, error) {
				dials++
				return nil, errors.New("unexpected dial")
			}))

			_, err := connector.Fetch(context.Background(), operation.Fetch{
				Method: method, Resource: "ftp://127.0.0.1/file.txt",
			})
			if err == nil || !strings.Contains(err.Error(), "unsupported ftp fetch method") {
				t.Fatalf("Fetch(%q) error = %v, want unsupported-method error", method, err)
			}
			if dials != 0 {
				t.Fatalf("Fetch(%q) dial count = %d, want zero", method, dials)
			}
		})
	}
}

func TestFTPRejectsUnsupportedStoreMethodsBeforeDial(t *testing.T) {
	dials := 0
	connector := NewFTP(ftpDialerFunc(func(context.Context, string, string) (net.Conn, error) {
		dials++
		return nil, errors.New("unexpected dial")
	}))

	_, err := connector.Store(context.Background(), operation.Store{
		Method: http.MethodPost, Resource: "ftp://127.0.0.1/file.txt",
	})
	if !errors.Is(err, operation.ErrUnsupported) {
		t.Fatalf("Store() error = %v, want operation.ErrUnsupported", err)
	}
	if dials != 0 {
		t.Fatalf("Store() dial count = %d, want zero", dials)
	}
}

func TestFTPReplyCodesMapToSemanticOutcomes(t *testing.T) {
	for _, test := range []struct {
		code int
		want operation.Outcome
	}{
		{code: 530, want: operation.OutcomePermissionDenied},
		{code: 532, want: operation.OutcomePermissionDenied},
		{code: 550, want: operation.OutcomeNotFound},
		{code: 425, want: operation.OutcomeUpstreamFailure},
		{code: 451, want: operation.OutcomeUpstreamFailure},
	} {
		if got := ftpOutcome(test.code); got != test.want {
			t.Errorf("ftpOutcome(%d) = %d, want %d", test.code, got, test.want)
		}
	}
}

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
	if _, err := io.Copy(io.Discard, result.Body); err != nil {
		t.Fatal(err)
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
	if _, err := io.Copy(io.Discard, result.Body); err != nil {
		t.Fatal(err)
	}
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

func TestFTPFetchReturnsBeforeDataCompletes(t *testing.T) {
	for _, test := range []struct {
		name    string
		method  string
		command string
	}{
		{name: "retrieve", method: "GET", command: "RETR"},
		{name: "list", command: "LIST"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testFTPFetchReturnsBeforeDataCompletes(t, test.method, test.command)
		})
	}
}

func testFTPFetchReturnsBeforeDataCompletes(t *testing.T, method, command string) {
	t.Helper()
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		data: dataClient,
	}
	transferReady := make(chan struct{})
	releaseTransfer := make(chan struct{})
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := openFTPTestTransfer(reader, control, command); err != nil {
			return err
		}
		close(transferReady)
		<-releaseTransfer
		payload := strings.Repeat("streamed-data-", 1<<17)
		if _, err := io.WriteString(dataServer, payload); err != nil {
			return err
		}
		if err := dataServer.Close(); err != nil {
			return err
		}
		return writeFTPLine(control, "226 Transfer complete")
	})

	resultReady := make(chan struct {
		result operation.Result
		err    error
	}, 1)
	go func() {
		var result operation.Result
		var err error
		connector := NewFTP(dialer)
		if command == "LIST" {
			result, err = connector.List(context.Background(), operation.List{
				Resource: "ftp://127.0.0.1:21/large.txt",
			})
		} else {
			result, err = connector.Fetch(context.Background(), operation.Fetch{
				Method: method, Resource: "ftp://127.0.0.1:21/large.txt",
			})
		}
		resultReady <- struct {
			result operation.Result
			err    error
		}{result: result, err: err}
	}()
	<-transferReady
	var fetched operation.Result
	select {
	case outcome := <-resultReady:
		if outcome.err != nil {
			t.Fatal(outcome.err)
		}
		fetched = outcome.result
	case <-time.After(100 * time.Millisecond):
		close(releaseTransfer)
		_ = dataServer.Close()
		_ = controlServer.Close()
		<-resultReady
		t.Fatal("Fetch waited for the complete data payload")
	}
	close(releaseTransfer)
	payload, err := io.ReadAll(fetched.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer fetched.Body.Close()
	if got, want := len(payload), len("streamed-data-")*(1<<17); got != want {
		t.Fatalf("payload length = %d, want %d", got, want)
	}
	if err := <-scriptDone; err != nil {
		t.Fatal(err)
	}
}

func TestFTPFetchBodySurfacesFailedCompletion(t *testing.T) {
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
		if err := openFTPTestTransfer(reader, control, "RETR"); err != nil {
			return err
		}
		if _, err := io.WriteString(dataServer, "partial payload"); err != nil {
			return err
		}
		if err := dataServer.Close(); err != nil {
			return err
		}
		return writeFTPLine(control, "550 Transfer failed")
	})

	result, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(result.Body)
	defer result.Body.Close()
	if got, want := string(payload), "partial payload"; got != want {
		t.Fatalf("payload = %q, want %q", got, want)
	}
	if err == nil || !strings.Contains(err.Error(), "completion rejected with 550") {
		t.Fatalf("body error = %v, want failed FTP completion", err)
	}
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestFTPFetchBodyCloseReleasesTransferSockets(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	trackedControl := &ftpCloseTrackingConn{
		Conn: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		closed: make(chan struct{}),
	}
	trackedData := &ftpCloseTrackingConn{Conn: dataClient, closed: make(chan struct{})}
	dialer := &ftpScriptDialer{control: trackedControl, data: trackedData}
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := openFTPTestTransfer(reader, control, "RETR"); err != nil {
			return err
		}
		buffer := make([]byte, 1)
		_, err := dataServer.Read(buffer)
		if err == nil {
			return errors.New("data connection remained readable after body close")
		}
		return nil
	})

	result, err := NewFTP(dialer).Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	for name, closed := range map[string]<-chan struct{}{
		"control": trackedControl.closed,
		"data":    trackedData.closed,
	} {
		select {
		case <-closed:
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("%s connection was not closed", name)
		}
	}
	_ = dataServer.Close()
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestFTPFetchBodyCancellationInterruptsRead(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		data: dataClient,
	}
	transferReady := make(chan struct{})
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := openFTPTestTransfer(reader, control, "RETR"); err != nil {
			return err
		}
		close(transferReady)
		buffer := make([]byte, 1)
		_, _ = dataServer.Read(buffer)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	result, err := NewFTP(dialer).Fetch(ctx, operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	<-transferReady
	readDone := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1)
		_, readErr := result.Body.Read(buffer)
		readDone <- readErr
	}()
	cancel()
	select {
	case readErr := <-readDone:
		if !errors.Is(readErr, context.Canceled) {
			t.Fatalf("body read error = %v, want context cancellation", readErr)
		}
	case <-time.After(250 * time.Millisecond):
		_ = result.Body.Close()
		t.Fatal("body read did not stop after cancellation")
	}
	_ = result.Body.Close()
	_ = dataServer.Close()
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestFTPFetchBodyUsesDataTimeout(t *testing.T) {
	controlClient, controlServer := net.Pipe()
	dataClient, dataServer := net.Pipe()
	defer dataServer.Close()
	dialer := &ftpScriptDialer{
		control: ftpRemoteAddrConn{
			Conn: controlClient,
			peer: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 21},
		},
		data: dataClient,
	}
	scriptDone := runFTPControlScript(controlServer, func(reader *bufio.Reader, control net.Conn) error {
		if err := openFTPTestTransfer(reader, control, "RETR"); err != nil {
			return err
		}
		buffer := make([]byte, 1)
		_, _ = dataServer.Read(buffer)
		return nil
	})
	connector := NewFTP(dialer)
	connector.dataTimeout = 25 * time.Millisecond
	result, err := connector.Fetch(context.Background(), operation.Fetch{
		Method:   "GET",
		Resource: "ftp://127.0.0.1:21/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err = result.Body.Read(make([]byte, 1))
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("body read error = %v, want data timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("stalled data read returned after %s, want bounded return", elapsed)
	}
	_ = result.Body.Close()
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
	}
}

func TestFTPStoreMapsFailedCompletionToSemanticOutcome(t *testing.T) {
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
		if err := openFTPTestTransfer(reader, control, "STOR"); err != nil {
			return err
		}
		if _, err := io.ReadAll(dataServer); err != nil {
			return err
		}
		return writeFTPLine(control, "550 Store failed")
	})

	result, err := NewFTP(dialer).Store(context.Background(), operation.Store{
		Method:   "PUT",
		Resource: "ftp://127.0.0.1:21/file.txt",
		Body:     strings.NewReader("upload"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != operation.OutcomeNotFound || result.Status != 0 {
		t.Fatalf("Store() result = %#v, want semantic not-found outcome", result)
	}
	if scriptErr := <-scriptDone; scriptErr != nil {
		t.Fatal(scriptErr)
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

func openFTPTestTransfer(reader *bufio.Reader, control net.Conn, command string) error {
	if err := ftpLoginScript(reader, control); err != nil {
		return err
	}
	if err := expectFTPCommand(reader, "EPSV"); err != nil {
		return err
	}
	if err := writeFTPLine(control, "229 Entering Extended Passive Mode (|||4242|)"); err != nil {
		return err
	}
	if err := expectFTPCommand(reader, command); err != nil {
		return err
	}
	return writeFTPLine(control, "150 Opening data connection")
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
