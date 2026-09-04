package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
