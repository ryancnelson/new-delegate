// Package tlsconfig defines side-effect-free TLS policy for the two sides of
// a gateway. Runtime adapters load the referenced certificate material.
package tlsconfig

import (
	"fmt"
	"strings"
)

// Frontend describes TLS termination for a listener.
type Frontend struct {
	CertificateFile string `json:"certificate_file" toml:"certificate_file"`
	PrivateKeyFile  string `json:"private_key_file" toml:"private_key_file"`
	MinimumVersion  string `json:"minimum_version,omitempty" toml:"minimum_version"`
}

// Validate checks frontend TLS policy without reading referenced files.
func (f Frontend) Validate() error {
	if strings.TrimSpace(f.CertificateFile) == "" && strings.TrimSpace(f.PrivateKeyFile) == "" {
		return fmt.Errorf("frontend TLS certificate and private key are required")
	}
	if (strings.TrimSpace(f.CertificateFile) == "") != (strings.TrimSpace(f.PrivateKeyFile) == "") {
		return fmt.Errorf("frontend TLS certificate and private key must be configured together")
	}
	if err := validateFileReference("frontend TLS certificate", f.CertificateFile); err != nil {
		return err
	}
	if err := validateFileReference("frontend TLS private key", f.PrivateKeyFile); err != nil {
		return err
	}
	return validateMinimumVersion(f.MinimumVersion)
}

// Backend describes server verification and optional client identity for a
// TLS connection to a mounted backend. There is deliberately no insecure
// verification bypass.
type Backend struct {
	CAFile                string `json:"ca_file,omitempty" toml:"ca_file"`
	ServerName            string `json:"server_name,omitempty" toml:"server_name"`
	ClientCertificateFile string `json:"client_certificate_file,omitempty" toml:"client_certificate_file"`
	ClientPrivateKeyFile  string `json:"client_private_key_file,omitempty" toml:"client_private_key_file"`
	MinimumVersion        string `json:"minimum_version,omitempty" toml:"minimum_version"`
}

// Validate checks backend TLS policy without reading referenced files.
func (b Backend) Validate() error {
	if b.CAFile != "" {
		if err := validateFileReference("backend TLS CA", b.CAFile); err != nil {
			return err
		}
	}
	if strings.ContainsAny(b.ServerName, " \t\r\n/") {
		return fmt.Errorf("backend TLS server name %q is invalid", b.ServerName)
	}
	if (strings.TrimSpace(b.ClientCertificateFile) == "") != (strings.TrimSpace(b.ClientPrivateKeyFile) == "") {
		return fmt.Errorf("backend TLS client certificate and private key must be configured together")
	}
	if b.ClientCertificateFile != "" {
		if err := validateFileReference("backend TLS client certificate", b.ClientCertificateFile); err != nil {
			return err
		}
		if err := validateFileReference("backend TLS client private key", b.ClientPrivateKeyFile); err != nil {
			return err
		}
	}
	return validateMinimumVersion(b.MinimumVersion)
}

func validateMinimumVersion(version string) error {
	if version == "" || version == "1.2" || version == "1.3" {
		return nil
	}
	return fmt.Errorf("TLS minimum version %q must be 1.2 or 1.3", version)
}

func validateFileReference(name, reference string) error {
	if strings.TrimSpace(reference) == "" {
		return fmt.Errorf("%s file reference is required", name)
	}
	for _, character := range reference {
		if character == 0 || character == '\r' || character == '\n' {
			return fmt.Errorf("%s file reference is invalid", name)
		}
	}
	return nil
}
