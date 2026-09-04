package connector

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"gitea.local/ryan/new-delegate/operation"
)

// FTP executes minimal Fetch/Store operations against ftp:// backends.
// It currently implements anonymous login, binary mode, and passive mode.
type FTP struct {
	dialer contextDialer
}

// NewFTP constructs a connector for FTP backends.
func NewFTP(dialer contextDialer) *FTP {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return &FTP{dialer: dialer}
}

// Fetch retrieves one resource from an ftp backend.
func (f *FTP) Fetch(ctx context.Context, fetch operation.Fetch) (operation.Result, error) {
	return f.operate(ctx, fetch.Method, fetch.Resource, "", fetch.Body)
}

// Store writes one resource to an ftp backend.
func (f *FTP) Store(ctx context.Context, store operation.Store) (operation.Result, error) {
	return f.operate(ctx, store.Method, store.Resource, "", store.Body)
}

func (f *FTP) operate(ctx context.Context, method, resource string, _ string, body io.Reader) (operation.Result, error) {
	if f == nil || f.dialer == nil {
		return operation.Result{}, fmt.Errorf("ftp connector requires a dialer")
	}
	parsed, err := parseFTPResource(resource)
	if err != nil {
		return operation.Result{}, err
	}

	control, err := f.dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return operation.Result{}, fmt.Errorf("dial ftp control: %w", err)
	}
	defer control.Close()

	controlReader := textproto.NewReader(bufio.NewReader(control))
	controlWriter := textproto.NewWriter(bufio.NewWriter(control))
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
	data, err := f.dialer.DialContext(ctx, "tcp", pasv)
	if err != nil {
		return operation.Result{}, fmt.Errorf("dial ftp data: %w", err)
	}
	defer data.Close()

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

func readFTPReply(reader *textproto.Reader) (ftpReply, error) {
	line, err := reader.ReadLine()
	if err != nil {
		return ftpReply{}, err
	}
	if len(line) < 3 {
		return ftpReply{}, fmt.Errorf("ftp reply too short: %q", line)
	}
	code, parseErr := strconv.Atoi(line[:3])
	if parseErr != nil {
		return ftpReply{}, parseErr
	}
	return ftpReply{Code: code, Body: line}, nil
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
	reader *textproto.Reader,
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
