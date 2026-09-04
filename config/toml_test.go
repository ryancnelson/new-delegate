package config

import (
	"reflect"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestParseTOMLReturnsValidatedCanonicalConfig(t *testing.T) {
	input := `
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
client_ip_header = "X-Forwarded-For"
trusted_proxies = ["10.0.0.0/8", "192.168.0.0/16"]

[servers.tls]
certificate_file = "certs/frontend.pem"
private_key_file = "certs/frontend-key.pem"
minimum_version = "1.3"

[[mounts]]
path = "/*"
target = "https://backend.internal/*"
priority = 10
server = "public"
protocol = "http"

[mounts.tls]
ca_file = "certs/backend-ca.pem"
server_name = "backend.internal"
client_certificate_file = "certs/client.pem"
client_private_key_file = "certs/client-key.pem"
minimum_version = "1.2"

[[policies]]
effect = "permit"
priority = 20
source = "192.0.2.0/24"
protocol = "http"
destination = "backend.internal"
method = "GET"
mount = "/*"
`

	got, err := ParseTOML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}
	want := Config{
		Servers: []Server{{
			Name: "public", Protocol: "http", Listen: ":8080",
			ClientIPHeader: "X-Forwarded-For",
			TrustedProxies: []string{"10.0.0.0/8", "192.168.0.0/16"},
			TLS: &tlsconfig.Frontend{
				CertificateFile: "certs/frontend.pem", PrivateKeyFile: "certs/frontend-key.pem",
				MinimumVersion: "1.3",
			},
		}},
		Mounts: []mount.Mount{{
			Path: "/*", Target: "https://backend.internal/*", Priority: 10,
			Server: "public", Protocol: "http",
			TLS: &tlsconfig.Backend{
				CAFile: "certs/backend-ca.pem", ServerName: "backend.internal",
				ClientCertificateFile: "certs/client.pem", ClientPrivateKeyFile: "certs/client-key.pem",
				MinimumVersion: "1.2",
			},
		}},
		Policies: []policy.Rule{{
			Effect: policy.Permit, Priority: 20, Source: "192.0.2.0/24",
			Protocol: "http", Destination: "backend.internal", Method: "GET", Mount: "/*",
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseTOML() = %#v, want %#v", got, want)
	}
}

func TestParseTOMLRejectsUnknownFields(t *testing.T) {
	_, err := ParseTOML(strings.NewReader(`
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
surprise = "do-not-echo-this-value"
`))
	if err == nil || !strings.Contains(err.Error(), "surprise") {
		t.Fatalf("ParseTOML() error = %v, want unknown field surprise", err)
	}
	if strings.Contains(err.Error(), "do-not-echo-this-value") {
		t.Fatalf("ParseTOML() error echoed configuration value: %v", err)
	}
}

func TestParseTOMLHasNoInsecureTLSVerificationBypass(t *testing.T) {
	_, err := ParseTOML(strings.NewReader(`
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
[[mounts]]
path = "/*"
target = "https://backend.internal/*"
[mounts.tls]
insecure_skip_verify = true
`))
	if err == nil || !strings.Contains(err.Error(), "insecure_skip_verify") {
		t.Fatalf("ParseTOML() error = %v, want unknown insecure bypass", err)
	}
}

func TestParseTOMLRejectsMalformedInput(t *testing.T) {
	_, err := ParseTOML(strings.NewReader(`servers = [`))
	if err == nil {
		t.Fatal("ParseTOML() error = nil, want syntax error")
	}
}

func TestParseTOMLRejectsInvalidCanonicalConfig(t *testing.T) {
	_, err := ParseTOML(strings.NewReader(`
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"

[[mounts]]
path = "relative"
target = "http://backend.internal/"
`))
	if err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("ParseTOML() error = %v, want validation error", err)
	}
}

func TestParseTOMLRejectsNilReader(t *testing.T) {
	_, err := ParseTOML(nil)
	if err == nil {
		t.Fatal("ParseTOML(nil) error = nil, want error")
	}
}

func TestParseTOMLRequiresServer(t *testing.T) {
	_, err := ParseTOML(strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "server") {
		t.Fatalf("ParseTOML() error = %v, want missing server error", err)
	}
}
