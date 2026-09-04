package tlsruntime

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestLoadBackend(t *testing.T) {
	certificateFile, privateKeyFile := writeFrontendCertificate(t)
	loaded, err := LoadBackend(tlsconfig.Backend{
		CAFile:                certificateFile,
		ServerName:            "backend.internal",
		ClientCertificateFile: certificateFile,
		ClientPrivateKeyFile:  privateKeyFile,
		MinimumVersion:        "1.3",
	})
	if err != nil {
		t.Fatalf("LoadBackend() error = %v", err)
	}
	if loaded.RootCAs == nil || loaded.ServerName != "backend.internal" ||
		loaded.MinVersion != tls.VersionTLS13 || len(loaded.Certificates) != 1 {
		t.Fatalf("loaded backend TLS config = %#v", loaded)
	}

	defaults, err := LoadBackend(tlsconfig.Backend{})
	if err != nil {
		t.Fatal(err)
	}
	if defaults.RootCAs != nil || defaults.MinVersion != tls.VersionTLS12 {
		t.Fatalf("default backend TLS = %#v, want system roots and TLS 1.2", defaults)
	}
}

func TestLoadBackendRejectsInvalidCAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBackend(tlsconfig.Backend{CAFile: path})
	if err == nil || !strings.Contains(err.Error(), "CA certificates") {
		t.Fatalf("LoadBackend() error = %v, want invalid CA error", err)
	}
}
