package tlsruntime

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

// LoadBackend loads custom trust and optional client identity for one backend
// TLS policy. A nil RootCAs field deliberately preserves Go's system roots.
func LoadBackend(config tlsconfig.Backend) (*tls.Config, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	loaded := &tls.Config{
		ServerName: config.ServerName,
		MinVersion: minimumVersion(config.MinimumVersion),
	}
	if config.CAFile != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system CA certificates: %w", err)
		}
		contents, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read backend TLS CA certificates: %w", err)
		}
		if !roots.AppendCertsFromPEM(contents) {
			return nil, fmt.Errorf("load backend TLS CA certificates: no certificates found")
		}
		loaded.RootCAs = roots
	}
	if config.ClientCertificateFile != "" {
		identity, err := tls.LoadX509KeyPair(config.ClientCertificateFile, config.ClientPrivateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load backend TLS client identity: %w", err)
		}
		loaded.Certificates = []tls.Certificate{identity}
	}
	return loaded, nil
}

func minimumVersion(version string) uint16 {
	if version == "1.3" {
		return tls.VersionTLS13
	}
	return tls.VersionTLS12
}
