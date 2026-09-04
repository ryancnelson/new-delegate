package connector

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/operation"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestHTTPRoutesSelectsTransportFromMountPolicy(t *testing.T) {
	plainCalls := atomic.Int32{}
	plainURL, _, closePlain := startHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainCalls.Add(1)
		_, _ = io.WriteString(w, "plain")
	}), false)
	defer closePlain()
	secureCalls := atomic.Int32{}
	secureURL, secureCert, closeSecure := startHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secureCalls.Add(1)
		_, _ = io.WriteString(w, "secure")
	}), true)
	defer closeSecure()

	policy := tlsconfig.Backend{ServerName: "127.0.0.1", MinimumVersion: "1.2"}
	roots := x509.NewCertPool()
	roots.AddCert(secureCert)
	routes := NewHTTPRoutes(&http.Client{}, map[tlsconfig.Backend]*tls.Config{
		policy: {RootCAs: roots, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12},
	})
	defer routes.CloseIdleConnections()

	for _, tt := range []struct {
		name    string
		mapping mount.Mount
		url     string
		body    string
	}{
		{name: "default plaintext", mapping: mount.Mount{}, url: plainURL, body: "plain"},
		{name: "selected TLS policy", mapping: mount.Mount{TLS: &policy}, url: secureURL, body: "secure"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routes.FetchForMount(context.Background(), tt.mapping, operation.Fetch{
				Method: http.MethodGet, Resource: tt.url,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			if err != nil || string(body) != tt.body {
				t.Fatalf("body = %q, %v; want %q", body, err, tt.body)
			}
		})
	}

	unknown := tlsconfig.Backend{ServerName: "unknown.internal"}
	_, err := routes.FetchForMount(context.Background(), mount.Mount{TLS: &unknown}, operation.Fetch{
		Method: http.MethodGet, Resource: secureURL,
	})
	if err == nil || !strings.Contains(err.Error(), "no preloaded backend TLS transport") {
		t.Fatalf("unknown route error = %v", err)
	}
	if plainCalls.Load() != 1 || secureCalls.Load() != 1 {
		t.Fatalf("backend calls = plain %d, secure %d; unknown route must not connect", plainCalls.Load(), secureCalls.Load())
	}
}

func TestHTTPRoutesExecutesStore(t *testing.T) {
	backendURL, _, closeBackend := startHTTPTestServer(t, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if request.Method != http.MethodPut || request.URL.Path != "/objects/report" || string(body) != "contents" || request.ContentLength != 8 {
			t.Errorf("request = %s %s length=%d body=%q", request.Method, request.URL.Path, request.ContentLength, body)
		}
		response.WriteHeader(http.StatusCreated)
	}), false)
	defer closeBackend()
	routes := NewHTTPRoutes(nil, nil)
	result, err := routes.StoreForMount(context.Background(), mount.Mount{}, operation.Store{
		Method: http.MethodPut, Resource: backendURL + "/objects/report",
		Body: strings.NewReader("contents"), Size: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if result.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", result.Status, http.StatusCreated)
	}
}

func TestHTTPRoutesDispatchesFTPFetchAndStore(t *testing.T) {
	store := map[string][]byte{}
	address, closeServer := startFakeFTPServer(t, map[string][]byte{
		"/reports/plan.txt": []byte("line one"),
	}, store)
	defer closeServer()

	routes := NewHTTPRoutes(nil, nil)
	fetchResult, err := routes.FetchForMount(context.Background(), mount.Mount{
		Target: "ftp://" + address + "/reports/plan.txt",
	}, operation.Fetch{
		Method: http.MethodGet, Resource: "ftp://" + address + "/reports/plan.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fetchResult.Body.Close()
	fetchBody, err := io.ReadAll(fetchResult.Body)
	if err != nil || string(fetchBody) != "line one" {
		t.Fatalf("fetch body = %q, %v; want line one", fetchBody, err)
	}
	if fetchResult.Status != 226 {
		t.Fatalf("fetch status = %d, want 226", fetchResult.Status)
	}

	storePayload := "new-bytes"
	storeResult, err := routes.StoreForMount(context.Background(), mount.Mount{
		Target: "ftp://" + address + "/reports/plan.txt",
	}, operation.Store{
		Method: http.MethodPut, Resource: "ftp://" + address + "/reports/plan.txt",
		Body: strings.NewReader(storePayload),
	})
	if err != nil {
		t.Fatal(err)
	}
	if storeResult.Status != 226 {
		t.Fatalf("store status = %d, want 226", storeResult.Status)
	}
	if got := string(store["/reports/plan.txt"]); got != storePayload {
		t.Fatalf("stored body = %q, want %q", got, storePayload)
	}

	listResult, err := routes.FetchForMount(context.Background(), mount.Mount{
		Target: "ftp://" + address + "/",
	}, operation.Fetch{
		Method:   "LIST",
		Resource: "ftp://" + address + "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer listResult.Body.Close()
	listBody, err := io.ReadAll(listResult.Body)
	if err != nil || string(listBody) != "reports/plan.txt\n" {
		t.Fatalf("list body = %q, %v; want reports/plan.txt\\n", listBody, err)
	}
	if listResult.Status != 226 {
		t.Fatalf("list status = %d, want 226", listResult.Status)
	}
}

func startFakeFTPServer(t *testing.T, files map[string][]byte, uploads map[string][]byte) (string, func()) {
	t.Helper()
	control, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			connection, acceptErr := control.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer connection.Close()
				var dataListener net.Listener
				var dataLock sync.Mutex
				reader := bufio.NewReader(connection)
				_ = writeFTPLine(connection, "220 welcome")
				for {
					commandLine, readErr := reader.ReadString('\n')
					if readErr != nil {
						return
					}
					command, argument := parseFTPCommand(commandLine)
					switch command {
					case "USER":
						_ = writeFTPLine(connection, "331 Password required")
					case "PASS":
						_ = writeFTPLine(connection, "230 User logged in")
					case "TYPE":
						_ = writeFTPLine(connection, "200 Type set to I")
					case "PASV":
						if dataListener != nil {
							_ = dataListener.Close()
						}
						dataListener, err = net.Listen("tcp", "127.0.0.1:0")
						if err != nil {
							_ = writeFTPLine(connection, "425 Can't open data connection")
							return
						}
						ip, port, splitErr := net.SplitHostPort(dataListener.Addr().String())
						if splitErr != nil {
							_ = writeFTPLine(connection, "425 Bad data listener")
							return
						}
						host := net.ParseIP(ip).To4()
						p, _ := strconv.Atoi(port)
						_ = writeFTPLine(connection, fmt.Sprintf(
							"227 Entering Passive Mode (%d,%d,%d,%d,%d,%d)",
							host[0], host[1], host[2], host[3], p/256, p%256,
						))
					case "RETR":
						if dataListener == nil {
							_ = writeFTPLine(connection, "425 Use PASV first")
							continue
						}
						dataConnection, acceptErr := dataListener.Accept()
						if acceptErr != nil {
							_ = writeFTPLine(connection, "425 Data connection failed")
							continue
						}
						_ = writeFTPLine(connection, "150 Opening data connection")
						dataLock.Lock()
						payload := files[argument]
						dataLock.Unlock()
						_, _ = dataConnection.Write(payload)
						_ = dataConnection.Close()
						_ = dataListener.Close()
						dataListener = nil
						_ = writeFTPLine(connection, "226 Transfer complete")
					case "STOR":
						if dataListener == nil {
							_ = writeFTPLine(connection, "425 Use PASV first")
							continue
						}
						_ = writeFTPLine(connection, "150 Opened data connection")
						dataConnection, acceptErr := dataListener.Accept()
						if acceptErr != nil {
							_ = writeFTPLine(connection, "425 Data connection failed")
							continue
						}
						received, readErr := io.ReadAll(dataConnection)
						_ = dataConnection.Close()
						_ = dataListener.Close()
						dataListener = nil
						if readErr != nil {
							_ = writeFTPLine(connection, "451 Read failed")
							continue
						}
						dataLock.Lock()
						uploads[argument] = received
						dataLock.Unlock()
						_ = writeFTPLine(connection, "226 Transfer complete")
					case "LIST":
						if dataListener == nil {
							_ = writeFTPLine(connection, "425 Use PASV first")
							continue
						}
						_ = writeFTPLine(connection, "150 Here comes the directory listing")
						dataConnection, acceptErr := dataListener.Accept()
						if acceptErr != nil {
							_ = writeFTPLine(connection, "425 Data connection failed")
							continue
						}
						dataLock.Lock()
						lines := make([]string, 0, len(files))
						for name := range files {
							lines = append(lines, strings.TrimPrefix(name, "/"))
						}
						dataLock.Unlock()
						sort.Strings(lines)
						for _, name := range lines {
							_, _ = dataConnection.Write([]byte(name + "\n"))
						}
						_ = dataConnection.Close()
						_ = dataListener.Close()
						dataListener = nil
						_ = writeFTPLine(connection, "226 Transfer complete")
					case "QUIT":
						_ = writeFTPLine(connection, "221 Bye")
						return
					default:
						_ = writeFTPLine(connection, "502 Command not implemented")
					}
				}
			}()
		}
	}()
	return control.Addr().String(), func() {
		_ = control.Close()
	}
}

func startHTTPTestServer(t *testing.T, handler http.Handler, secure bool) (string, *x509.Certificate, func()) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	closeServer := func() {
		_ = server.Shutdown(context.Background())
	}
	addr := listener.Addr().String()

	if secure {
		cert, parsedCert := mustGenerateSelfSignedCert(t, addr)
		tlsListener := tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"http/1.1"},
		})
		go func() {
			if err := server.Serve(tlsListener); err != nil && !isErrServerClosed(err) {
				t.Error(err)
			}
		}()
		return "https://" + addr, parsedCert, closeServer
	}

	go func() {
		if err := server.Serve(listener); err != nil && !isErrServerClosed(err) {
			t.Error(err)
		}
	}()
	return "http://" + addr, nil, closeServer
}

func isErrServerClosed(err error) bool {
	return err != nil && (err == http.ErrServerClosed)
}

func mustGenerateSelfSignedCert(t *testing.T, listenerAddr string) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	host, _, splitErr := net.SplitHostPort(listenerAddr)
	if splitErr != nil {
		t.Fatal(splitErr)
	}
	ip := net.ParseIP(host)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	serial := big.NewInt(0)
	serial.SetInt64(time.Now().UnixNano())
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"new-delegate test"}},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	if ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	parsedCert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert, parsedCert
}

func parseFTPCommand(line string) (string, string) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return "", ""
	}
	if len(fields) == 1 {
		return strings.ToUpper(fields[0]), ""
	}
	return strings.ToUpper(fields[0]), fields[1]
}

func writeFTPLine(connection net.Conn, line string) error {
	_, err := connection.Write([]byte(line + "\r\n"))
	return err
}
