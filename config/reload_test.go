package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestReloadTOMLFilePublishesValidRoutingChange(t *testing.T) {
	store, err := NewStore(storeTestConfig("old.internal", 8080))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	text := strings.ReplaceAll(reloadTOML, "HOST", "new.internal")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTOMLFile(store, path); err != nil {
		t.Fatalf("ReloadTOMLFile() error = %v", err)
	}
	if got := store.Snapshot().Mounts[0].Target; got != "http://new.internal/*" {
		t.Fatalf("target = %q, want new backend", got)
	}
}

func TestReloadTOMLFileRejectsBackendTLSPolicyChange(t *testing.T) {
	initial := storeTestConfig("backend.internal", 8080)
	initial.Mounts[0].Target = "https://backend.internal/*"
	initial.Mounts[0].TLS = &tlsconfig.Backend{CAFile: "old-ca.pem"}
	store, err := NewStore(initial)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	text := `
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
[[mounts]]
path = "/*"
target = "https://backend.internal/*"
[mounts.tls]
ca_file = "new-ca.pem"
[[policies]]
effect = "permit"
protocol = "http"
destination = "backend.internal"
source = "*"
`
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTOMLFile(store, path); err == nil || !strings.Contains(err.Error(), "backend TLS") {
		t.Fatalf("reload error = %v, want backend TLS restart requirement", err)
	}
	if got := store.Snapshot().Mounts[0].TLS.CAFile; got != "old-ca.pem" {
		t.Fatalf("rejected reload published CA file %q", got)
	}
}

func TestReloadTOMLFileRollsBackInvalidCandidate(t *testing.T) {
	initial := storeTestConfig("old.internal", 8080)
	store, err := NewStore(initial)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	if err := os.WriteFile(path, []byte("not valid = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTOMLFile(store, path); err == nil {
		t.Fatal("reload error = nil")
	}
	if got := store.Snapshot().Mounts[0].Target; got != initial.Mounts[0].Target {
		t.Fatalf("target changed after invalid reload: %q", got)
	}
}

func TestReloadTOMLFileRejectsListenerTopologyChange(t *testing.T) {
	store, err := NewStore(storeTestConfig("old.internal", 8080))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	text := strings.ReplaceAll(strings.ReplaceAll(reloadTOML, "HOST", "new.internal"), ":8080", ":8081")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTOMLFile(store, path); err == nil || !strings.Contains(err.Error(), "listener topology") {
		t.Fatalf("reload error = %v, want topology error", err)
	}
	if got := store.Snapshot().Servers[0].Listen; got != ":8080" {
		t.Fatalf("listen changed to %q", got)
	}
}

func TestReloadTOMLFileAllowsTrustedProxyPolicyChange(t *testing.T) {
	initial := storeTestConfig("old.internal", 8080)
	initial.Servers[0].ClientIPHeader = "X-Forwarded-For"
	initial.Servers[0].TrustedProxies = []string{"10.0.0.0/8"}
	store, err := NewStore(initial)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	text := strings.ReplaceAll(reloadTOML, "HOST", "old.internal")
	text = strings.Replace(text, "listen = \":8080\"", `listen = ":8080"
client_ip_header = "X-Forwarded-For"
trusted_proxies = ["192.168.0.0/16"]`, 1)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTOMLFile(store, path); err != nil {
		t.Fatalf("ReloadTOMLFile() error = %v", err)
	}
	if got := store.Snapshot().Servers[0].TrustedProxies; len(got) != 1 || got[0] != "192.168.0.0/16" {
		t.Fatalf("trusted proxies = %v, want reloaded policy", got)
	}
}

const reloadTOML = `
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
[[mounts]]
path = "/*"
target = "http://HOST/*"
[[policies]]
effect = "permit"
protocol = "http"
destination = "HOST"
source = "*"
`
