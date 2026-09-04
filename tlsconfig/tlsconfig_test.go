package tlsconfig

import (
	"strings"
	"testing"
)

func TestFrontendValidate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		config  Frontend
		wantErr string
	}{
		{
			name: "certificate and key",
			config: Frontend{
				CertificateFile: "certs/frontend.pem",
				PrivateKeyFile:  "certs/frontend-key.pem",
				MinimumVersion:  "1.3",
			},
		},
		{name: "missing identity", config: Frontend{}, wantErr: "certificate"},
		{name: "missing private key", config: Frontend{CertificateFile: "cert.pem"}, wantErr: "together"},
		{name: "missing certificate", config: Frontend{PrivateKeyFile: "key.pem"}, wantErr: "together"},
		{
			name: "old minimum version",
			config: Frontend{
				CertificateFile: "cert.pem", PrivateKeyFile: "key.pem", MinimumVersion: "1.1",
			},
			wantErr: "minimum",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestBackendValidate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		config  Backend
		wantErr string
	}{
		{
			name: "custom trust and client identity",
			config: Backend{
				CAFile:                "certs/internal-ca.pem",
				ServerName:            "backend.internal",
				ClientCertificateFile: "certs/client.pem",
				ClientPrivateKeyFile:  "certs/client-key.pem",
				MinimumVersion:        "1.2",
			},
		},
		{name: "system trust defaults", config: Backend{}},
		{name: "missing client key", config: Backend{ClientCertificateFile: "client.pem"}, wantErr: "together"},
		{name: "missing client certificate", config: Backend{ClientPrivateKeyFile: "key.pem"}, wantErr: "together"},
		{name: "invalid server name", config: Backend{ServerName: "backend internal"}, wantErr: "server name"},
		{name: "invalid minimum version", config: Backend{MinimumVersion: "latest"}, wantErr: "minimum"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
