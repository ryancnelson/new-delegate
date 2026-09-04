package tlsruntime

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestLoadFrontend(t *testing.T) {
	certificateFile, privateKeyFile := writeFrontendCertificate(t)
	loaded, err := LoadFrontend(tlsconfig.Frontend{
		CertificateFile: certificateFile,
		PrivateKeyFile:  privateKeyFile,
		MinimumVersion:  "1.3",
	})
	if err != nil {
		t.Fatalf("LoadFrontend() error = %v", err)
	}
	if loaded.MinVersion != tls.VersionTLS13 || len(loaded.Certificates) != 1 {
		t.Fatalf("loaded TLS config = %#v, want one certificate and TLS 1.3", loaded)
	}

	loaded, err = LoadFrontend(tlsconfig.Frontend{CertificateFile: certificateFile, PrivateKeyFile: privateKeyFile})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.MinVersion != tls.VersionTLS12 {
		t.Fatalf("default minimum = %x, want TLS 1.2", loaded.MinVersion)
	}
}

func TestLoadFrontendReportsUnreadableIdentity(t *testing.T) {
	_, err := LoadFrontend(tlsconfig.Frontend{CertificateFile: "missing.pem", PrivateKeyFile: "missing-key.pem"})
	if err == nil || !strings.Contains(err.Error(), "load frontend TLS identity") {
		t.Fatalf("LoadFrontend() error = %v, want identity load failure", err)
	}
}

func writeFrontendCertificate(t *testing.T) (string, string) {
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
	directory := t.TempDir()
	certificateFile := filepath.Join(directory, "certificate.pem")
	privateKeyFile := filepath.Join(directory, "private-key.pem")
	if err := os.WriteFile(certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certificateFile, privateKeyFile
}
