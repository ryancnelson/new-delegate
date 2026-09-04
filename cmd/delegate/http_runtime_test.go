package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/connector"
	gatewayserver "gitea.local/ryan/new-delegate/server"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestPrepareHTTPRuntimeServesPlaintextAndTLSWithMinimumVersion(t *testing.T) {
	certificateFile, privateKeyFile, roots := writeRuntimeCertificate(t)
	configured := config.Config{Servers: []config.Server{
		{Name: "plain", Protocol: "http", Listen: "127.0.0.1:10001"},
		{Name: "secure", Protocol: "http", Listen: "127.0.0.1:10002", TLS: &tlsconfig.Frontend{
			CertificateFile: certificateFile, PrivateKeyFile: privateKeyFile, MinimumVersion: "1.3",
		}},
	}}
	store, err := config.NewStore(configured)
	if err != nil {
		t.Fatal(err)
	}
	listen := func(_, _ string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	servers, listeners, err := prepareHTTPRuntime(
		configured, store.Snapshot, connector.NewHTTP(nil), listen,
	)
	if err != nil {
		t.Fatalf("prepareHTTPRuntime() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gatewayserver.ServeAll(ctx, servers, listeners, time.Second) }()

	plainResponse, err := (&http.Client{Timeout: time.Second}).Get("http://" + listeners[0].Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = plainResponse.Body.Close()
	if plainResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("plaintext status = %d, want handler response", plainResponse.StatusCode)
	}

	oldClient := runtimeTLSClient(roots, tls.VersionTLS12, tls.VersionTLS12)
	if response, err := oldClient.Get("https://" + listeners[1].Addr().String()); err == nil {
		_ = response.Body.Close()
		t.Fatal("TLS 1.2 client reached TLS 1.3 listener")
	}
	secureClient := runtimeTLSClient(roots, tls.VersionTLS13, tls.VersionTLS13)
	secureResponse, err := secureClient.Get("https://" + listeners[1].Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = secureResponse.Body.Close()
	if secureResponse.StatusCode != http.StatusNotFound || secureResponse.TLS == nil || secureResponse.TLS.Version != tls.VersionTLS13 {
		t.Fatalf("secure response = status %d, TLS %#v", secureResponse.StatusCode, secureResponse.TLS)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeAll() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("mixed listener group did not stop")
	}
}

func TestPrepareHTTPRuntimeLoadsEveryCertificateBeforeBinding(t *testing.T) {
	configured := config.Config{Servers: []config.Server{
		{Name: "plain", Protocol: "http", Listen: "127.0.0.1:10001"},
		{Name: "broken", Protocol: "http", Listen: "127.0.0.1:10002", TLS: &tlsconfig.Frontend{
			CertificateFile: "missing.pem", PrivateKeyFile: "missing-key.pem",
		}},
	}}
	store, err := config.NewStore(configured)
	if err != nil {
		t.Fatal(err)
	}
	bindCalls := 0
	_, _, err = prepareHTTPRuntime(configured, store.Snapshot, connector.NewHTTP(nil), func(_, _ string) (net.Listener, error) {
		bindCalls++
		return nil, errors.New("unexpected bind")
	})
	if err == nil || !strings.Contains(err.Error(), "load TLS for server") {
		t.Fatalf("prepareHTTPRuntime() error = %v, want certificate failure", err)
	}
	if bindCalls != 0 {
		t.Fatalf("bind calls = %d, want zero before all identities load", bindCalls)
	}
}

func TestPrepareHTTPRuntimeClosesEarlierListenerOnBindFailure(t *testing.T) {
	configured := config.Config{Servers: []config.Server{
		{Name: "one", Protocol: "http", Listen: "127.0.0.1:10001"},
		{Name: "two", Protocol: "http", Listen: "127.0.0.1:10002"},
	}}
	store, err := config.NewStore(configured)
	if err != nil {
		t.Fatal(err)
	}
	var first net.Listener
	calls := 0
	listen := func(_, _ string) (net.Listener, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("second bind failed")
		}
		first, err = net.Listen("tcp", "127.0.0.1:0")
		return first, err
	}
	_, _, err = prepareHTTPRuntime(configured, store.Snapshot, connector.NewHTTP(nil), listen)
	if err == nil || !strings.Contains(err.Error(), "second bind failed") {
		t.Fatalf("prepareHTTPRuntime() error = %v", err)
	}
	if closeErr := first.Close(); !errors.Is(closeErr, net.ErrClosed) {
		t.Fatalf("first listener remained open; second close error = %v", closeErr)
	}
}

func runtimeTLSClient(roots *x509.CertPool, minimum, maximum uint16) *http.Client {
	return &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs: roots, MinVersion: minimum, MaxVersion: maximum,
		}},
	}
}

func writeRuntimeCertificate(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	directory := t.TempDir()
	certificateFile := filepath.Join(directory, "certificate.pem")
	privateKeyFile := filepath.Join(directory, "private-key.pem")
	if err := os.WriteFile(certificateFile, certificatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificatePEM) {
		t.Fatal("append generated root certificate")
	}
	return certificateFile, privateKeyFile, roots
}
