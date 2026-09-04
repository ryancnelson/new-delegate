package connector

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitea.local/ryan/new-delegate/operation"
)

// FTP executes minimal Fetch/Store operations against ftp:// backends.
// It currently implements anonymous login, binary mode, and passive mode.
type FTP struct {
	dialer         contextDialer
	dialTimeout    time.Duration
	controlTimeout time.Duration
	dataTimeout    time.Duration
	maxReplyBytes  int
}

const (
	defaultFTPDialTimeout    = 10 * time.Second
	defaultFTPControlTimeout = 30 * time.Second
	defaultFTPDataTimeout    = 2 * time.Minute
	defaultFTPMaxReplyBytes  = 64 << 10
)

// NewFTP constructs a connector for FTP backends.
func NewFTP(dialer contextDialer) *FTP {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return &FTP{
		dialer:         dialer,
		dialTimeout:    defaultFTPDialTimeout,
		controlTimeout: defaultFTPControlTimeout,
		dataTimeout:    defaultFTPDataTimeout,
		maxReplyBytes:  defaultFTPMaxReplyBytes,
	}
}

// Fetch retrieves one resource from an ftp backend.
func (f *FTP) Fetch(ctx context.Context, fetch operation.Fetch) (operation.Result, error) {
	return f.operate(ctx, fetch.Method, fetch.Resource, "", fetch.Body)
}

// Store writes one resource to an ftp backend.
func (f *FTP) Store(ctx context.Context, store operation.Store) (operation.Result, error) {
	return f.operate(ctx, store.Method, store.Resource, "", store.Body)
}

func (f *FTP) operate(ctx context.Context, method, resource string, _ string, body io.Reader) (result operation.Result, resultErr error) {
	defer func() {
		if resultErr != nil && ctx.Err() != nil {
			resultErr = ctx.Err()
		}
	}()
	if f == nil || f.dialer == nil {
		return operation.Result{}, fmt.Errorf("ftp connector requires a dialer")
	}
	parsed, err := parseFTPResource(resource)
	if err != nil {
		return operation.Result{}, err
	}

	control, err := f.dial(ctx, parsed.Host)
	if err != nil {
		return operation.Result{}, fmt.Errorf("dial ftp control: %w", err)
	}
	defer control.Close()
	stopControlCancellation := context.AfterFunc(ctx, func() { _ = control.Close() })
	defer stopControlCancellation()

	controlIO := &ftpDeadlineConn{Conn: control, timeout: f.controlTimeoutValue()}
	controlReader := newFTPReplyReader(controlIO, f.maxReplyBytesValue())
	controlWriter := textproto.NewWriter(bufio.NewWriter(controlIO))
	if _, err := readFTPReply(controlReader); err != nil {
		return operation.Result{}, fmt.Errorf("connect ftp: %w", err)
	}
	if err := writeFTPCommand(controlWriter, "USER", "anonymous"); err != nil {
		return operation.Result{}, err
	}
	reply, err := readFTPReply(controlReader)
	if err != nil {
		return operation.Result{}, fmt.Errorf("ftp USER: %w", err)
	}
	if reply.Code != 331 && reply.Code != 230 {
		return operation.Result{}, fmt.Errorf("ftp USER rejected with %d", reply.Code)
	}
	if reply.Code == 331 {
		if err := writeFTPCommand(controlWriter, "PASS", "anonymous@example.com"); err != nil {
			return operation.Result{}, err
		}
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp PASS: %w", err)
		}
		if reply.Code != 230 {
			return operation.Result{}, fmt.Errorf("ftp PASS rejected with %d", reply.Code)
		}
	}
	if err := writeFTPCommand(controlWriter, "TYPE", "I"); err != nil {
		return operation.Result{}, err
	}
	reply, err = readFTPReply(controlReader)
	if err != nil {
		return operation.Result{}, fmt.Errorf("ftp TYPE: %w", err)
	}
	if reply.Code != 200 {
		return operation.Result{}, fmt.Errorf("ftp TYPE rejected with %d", reply.Code)
	}

	pasv, err := ftpPassiveAddress(control, controlReader, controlWriter)
	if err != nil {
		return operation.Result{}, err
	}
	data, err := f.dial(ctx, pasv)
	if err != nil {
		return operation.Result{}, fmt.Errorf("dial ftp data: %w", err)
	}
	defer data.Close()
	stopDataCancellation := context.AfterFunc(ctx, func() { _ = data.Close() })
	defer stopDataCancellation()
	data = &ftpDeadlineConn{Conn: data, timeout: f.dataTimeoutValue()}

	command := strings.ToUpper(method)
	switch command {
	case "PUT":
		command = "STOR"
	case "LIST":
		command = "LIST"
	case http.MethodGet:
		command = "RETR"
	default:
		command = "RETR"
	}

	switch command {
	case "STOR":
		if err := writeFTPCommand(controlWriter, command, parsed.Path); err != nil {
			return operation.Result{}, err
		}
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp STOR: %w", err)
		}
		if reply.Code < 100 || reply.Code > 199 {
			return operation.Result{}, fmt.Errorf("ftp STOR rejected with %d", reply.Code)
		}
		if body == nil {
			body = strings.NewReader("")
		}
		if _, err := io.Copy(data, body); err != nil {
			return operation.Result{}, fmt.Errorf("ftp STOR data copy: %w", err)
		}
		_ = data.Close()
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp STOR completion: %w", err)
		}
		return operation.Result{Status: reply.Code}, nil
	case "RETR":
		if err := writeFTPCommand(controlWriter, command, parsed.Path); err != nil {
			return operation.Result{}, err
		}
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp RETR: %w", err)
		}
		if reply.Code < 100 || reply.Code > 199 {
			return operation.Result{}, fmt.Errorf("ftp RETR rejected with %d", reply.Code)
		}
		payload, err := io.ReadAll(data)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp RETR data read: %w", err)
		}
		_ = data.Close()
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp RETR completion: %w", err)
		}
		return operation.Result{Status: reply.Code, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	case "LIST":
		if err := writeFTPCommand(controlWriter, command, parsed.Path); err != nil {
			return operation.Result{}, err
		}
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp LIST: %w", err)
		}
		if reply.Code < 100 || reply.Code > 199 {
			return operation.Result{}, fmt.Errorf("ftp LIST rejected with %d", reply.Code)
		}
		payload, err := io.ReadAll(data)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp LIST data read: %w", err)
		}
		_ = data.Close()
		reply, err = readFTPReply(controlReader)
		if err != nil {
			return operation.Result{}, fmt.Errorf("ftp LIST completion: %w", err)
		}
		return operation.Result{Status: reply.Code, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	}
	return operation.Result{}, fmt.Errorf("unsupported ftp command %q", command)
}

type ftpReply struct {
	Code int
	Body string
}

type ftpReplyReader struct {
	reader   *bufio.Reader
	maxBytes int
}

func newFTPReplyReader(reader io.Reader, maxBytes int) *ftpReplyReader {
	if maxBytes <= 0 {
		maxBytes = defaultFTPMaxReplyBytes
	}
	return &ftpReplyReader{
		reader:   bufio.NewReaderSize(reader, maxBytes+1),
		maxBytes: maxBytes,
	}
}

func (r *ftpReplyReader) ReadReply() (ftpReply, error) {
	line, used, err := r.readLine(r.maxBytes)
	if err != nil {
		return ftpReply{}, err
	}
	code, separator, err := parseFTPReplyStart(line)
	if err != nil {
		return ftpReply{}, err
	}
	lines := []string{line}
	if separator != '-' {
		return ftpReply{Code: code, Body: line}, nil
	}

	terminator := fmt.Sprintf("%03d ", code)
	for {
		line, consumed, readErr := r.readLine(r.maxBytes - used)
		if readErr != nil {
			return ftpReply{}, fmt.Errorf("ftp multiline reply: %w", readErr)
		}
		used += consumed
		lines = append(lines, line)
		if strings.HasPrefix(line, terminator) {
			return ftpReply{Code: code, Body: strings.Join(lines, "\n")}, nil
		}
	}
}

func (r *ftpReplyReader) readLine(remaining int) (string, int, error) {
	if remaining <= 0 {
		return "", 0, fmt.Errorf("ftp reply exceeds %d bytes", r.maxBytes)
	}
	raw, err := r.reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) || len(raw) > remaining {
		return "", 0, fmt.Errorf("ftp reply exceeds %d bytes", r.maxBytes)
	}
	if err != nil {
		return "", 0, err
	}
	line := strings.TrimSuffix(string(raw), "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, len(raw), nil
}

func parseFTPReplyStart(line string) (int, byte, error) {
	if len(line) < 3 {
		return 0, 0, fmt.Errorf("ftp reply too short: %q", line)
	}
	code, err := strconv.Atoi(line[:3])
	if err != nil || code < 100 || code > 599 {
		return 0, 0, fmt.Errorf("ftp reply has invalid code: %q", line)
	}
	if len(line) == 3 {
		return code, ' ', nil
	}
	if line[3] != ' ' && line[3] != '-' {
		return 0, 0, fmt.Errorf("ftp reply has invalid separator: %q", line)
	}
	return code, line[3], nil
}

func readFTPReply(reader *ftpReplyReader) (ftpReply, error) {
	return reader.ReadReply()
}

func writeFTPCommand(writer *textproto.Writer, command string, args ...string) error {
	line := command
	if len(args) != 0 && strings.TrimSpace(args[0]) != "" {
		line = line + " " + strings.Join(args, " ")
	}
	if err := writer.PrintfLine("%s", line); err != nil {
		return err
	}
	return writer.W.Flush()
}

func ftpPassiveAddress(
	control net.Conn,
	reader *ftpReplyReader,
	writer *textproto.Writer,
) (string, error) {
	peer, err := ftpControlPeerIP(control)
	if err != nil {
		return "", err
	}
	if err := writeFTPCommand(writer, "EPSV"); err != nil {
		return "", err
	}
	reply, err := readFTPReply(reader)
	if err != nil {
		return "", fmt.Errorf("ftp EPSV: %w", err)
	}
	if reply.Code == 229 {
		port, parseErr := parseEPSVPort(reply.Body)
		if parseErr != nil {
			return "", parseErr
		}
		return net.JoinHostPort(peer, strconv.Itoa(port)), nil
	}
	if !ftpEPSVUnsupported(reply.Code) {
		return "", fmt.Errorf("ftp EPSV rejected with %d", reply.Code)
	}

	if err := writeFTPCommand(writer, "PASV"); err != nil {
		return "", err
	}
	reply, err = readFTPReply(reader)
	if err != nil {
		return "", fmt.Errorf("ftp PASV: %w", err)
	}
	if reply.Code != 227 {
		return "", fmt.Errorf("ftp PASV rejected with %d", reply.Code)
	}
	port, err := parsePASVPort(reply.Body)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(peer, strconv.Itoa(port)), nil
}

type ftpDeadlineConn struct {
	net.Conn
	timeout time.Duration
}

func (c *ftpDeadlineConn) Read(buffer []byte) (int, error) {
	if c.timeout > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	}
	return c.Conn.Read(buffer)
}

func (c *ftpDeadlineConn) Write(buffer []byte) (int, error) {
	if c.timeout > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	return c.Conn.Write(buffer)
}

func (f *FTP) dial(ctx context.Context, address string) (net.Conn, error) {
	timeout := f.dialTimeout
	if timeout <= 0 {
		timeout = defaultFTPDialTimeout
	}
	dialContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f.dialer.DialContext(dialContext, "tcp", address)
}

func (f *FTP) controlTimeoutValue() time.Duration {
	if f.controlTimeout > 0 {
		return f.controlTimeout
	}
	return defaultFTPControlTimeout
}

func (f *FTP) dataTimeoutValue() time.Duration {
	if f.dataTimeout > 0 {
		return f.dataTimeout
	}
	return defaultFTPDataTimeout
}

func (f *FTP) maxReplyBytesValue() int {
	if f.maxReplyBytes > 0 {
		return f.maxReplyBytes
	}
	return defaultFTPMaxReplyBytes
}

func ftpControlPeerIP(control net.Conn) (string, error) {
	if control == nil || control.RemoteAddr() == nil {
		return "", fmt.Errorf("ftp control connection has no remote address")
	}
	host, _, err := net.SplitHostPort(control.RemoteAddr().String())
	if err != nil {
		return "", fmt.Errorf("ftp control peer address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("ftp control peer is not an IP address: %q", host)
	}
	return ip.String(), nil
}

func ftpEPSVUnsupported(code int) bool {
	switch code {
	case 500, 501, 502, 522:
		return true
	default:
		return false
	}
}

func parseEPSVPort(body string) (int, error) {
	inside, err := ftpParenthesizedReply(body, "EPSV")
	if err != nil {
		return 0, err
	}
	if len(inside) < 5 {
		return 0, fmt.Errorf("ftp EPSV response malformed: %q", body)
	}
	delimiter := inside[:1]
	parts := strings.Split(inside, delimiter)
	if len(parts) != 5 || parts[0] != "" || parts[1] != "" || parts[2] != "" || parts[4] != "" {
		return 0, fmt.Errorf("ftp EPSV response malformed: %q", body)
	}
	return parseFTPPort(parts[3], "EPSV", body)
}

func parsePASVPort(body string) (int, error) {
	inside, err := ftpParenthesizedReply(body, "PASV")
	if err != nil {
		return 0, err
	}
	parts := strings.Split(inside, ",")
	if len(parts) != 6 {
		return 0, fmt.Errorf("ftp PASV response malformed: %q", body)
	}
	values := make([]int, len(parts))
	for index, part := range parts {
		value, parseErr := strconv.Atoi(strings.TrimSpace(part))
		if parseErr != nil || value < 0 || value > 255 {
			return 0, fmt.Errorf("ftp PASV response malformed: %q", body)
		}
		values[index] = value
	}
	port := values[4]*256 + values[5]
	if port == 0 {
		return 0, fmt.Errorf("ftp PASV response malformed: %q", body)
	}
	return port, nil
}

func ftpParenthesizedReply(body, command string) (string, error) {
	start := strings.Index(body, "(")
	end := strings.Index(body, ")")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("ftp %s response malformed: %q", command, body)
	}
	return body[start+1 : end], nil
}

func parseFTPPort(value, command, body string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("ftp %s response malformed: %q", command, body)
	}
	return port, nil
}

type ftpTarget struct {
	Host string
	Path string
}

func parseFTPResource(resource string) (ftpTarget, error) {
	parsed, err := url.Parse(resource)
	if err != nil {
		return ftpTarget{}, fmt.Errorf("parse ftp resource: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "ftp") {
		return ftpTarget{}, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return ftpTarget{}, fmt.Errorf("ftp resource requires host")
	}
	return ftpTarget{
		Host: parsed.Host,
		Path: normalizeFTPPath(parsed.Path),
	}, nil
}

func normalizeFTPPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
