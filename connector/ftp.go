package connector

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
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

	pasv, err := ftpPassiveAddress(controlReader, controlWriter)
	if err != nil {
		return operation.Result{}, err
	}
	data, err := f.dialer.DialContext(ctx, "tcp", pasv)
	if err != nil {
		return operation.Result{}, fmt.Errorf("dial ftp data: %w", err)
	}
	defer data.Close()

	switch strings.ToUpper(method) {
	case "PUT":
		if err := writeFTPCommand(controlWriter, "STOR", parsed.Path); err != nil {
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
	default:
		if err := writeFTPCommand(controlWriter, "RETR", parsed.Path); err != nil {
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
	}
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
	reader *textproto.Reader,
	writer *textproto.Writer,
) (string, error) {
	if err := writeFTPCommand(writer, "PASV"); err != nil {
		return "", err
	}
	reply, err := readFTPReply(reader)
	if err != nil {
		return "", fmt.Errorf("ftp PASV: %w", err)
	}
	if reply.Code != 227 {
		return "", fmt.Errorf("ftp PASV rejected with %d", reply.Code)
	}
	start := strings.Index(reply.Body, "(")
	end := strings.Index(reply.Body, ")")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("ftp PASV response malformed: %q", reply.Body)
	}
	parts := strings.Split(reply.Body[start+1:end], ",")
	if len(parts) != 6 {
		return "", fmt.Errorf("ftp PASV response malformed: %q", reply.Body)
	}
	portHigh, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return "", err
	}
	portLow, err := strconv.Atoi(strings.TrimSpace(parts[5]))
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(strings.TrimSpace(strings.Join(parts[:4], ".")), strconv.Itoa(portHigh*256+portLow)), nil
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
