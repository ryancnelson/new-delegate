// Package tlsruntime turns validated TLS policy into standard-library runtime
// values and performs the explicitly requested certificate file I/O.
package tlsruntime

import (
	"crypto/tls"
	"fmt"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

// LoadFrontend validates and loads a listener identity. The default minimum
// is TLS 1.2; callers must finish loading every configured identity before
// exposing any listener.
func LoadFrontend(config tlsconfig.Frontend) (*tls.Config, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	identity, err := tls.LoadX509KeyPair(config.CertificateFile, config.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load frontend TLS identity: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{identity},
		MinVersion:   minimumVersion(config.MinimumVersion),
	}, nil
}
