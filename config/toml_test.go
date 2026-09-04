package config

import (
	"reflect"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/policy"
)

func TestParseTOMLReturnsValidatedCanonicalConfig(t *testing.T) {
	input := `
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
client_ip_header = "X-Forwarded-For"
trusted_proxies = ["10.0.0.0/8", "192.168.0.0/16"]

[[mounts]]
path = "/*"
target = "http://backend.internal/*"
priority = 10
server = "public"
protocol = "http"

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
		}},
		Mounts: []mount.Mount{{
			Path: "/*", Target: "http://backend.internal/*", Priority: 10,
			Server: "public", Protocol: "http",
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
