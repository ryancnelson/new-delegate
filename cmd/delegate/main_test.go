package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitea.local/ryan/new-delegate/config"
	"gitea.local/ryan/new-delegate/mount"
	"gitea.local/ryan/new-delegate/tlsconfig"
)

func TestRunCheckPrintsCanonicalConfigWithoutServing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{
		"check",
		"SERVER=http",
		"-P8080",
		`MOUNT="/* http://backend.internal/*"`,
		`PERMIT="http:backend.internal:*"`,
	}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	for _, fragment := range []string{`"protocol": "http"`, `"listen": ":8080"`, `"effect": "permit"`} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout missing %q:\n%s", fragment, stdout.String())
		}
	}
}

func TestRunCheckLoadsTOMLConfigWithoutServing(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "delegate.toml")
	contents := []byte(`
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"

[[mounts]]
path = "/*"
target = "http://backend.internal/*"

[[policies]]
effect = "permit"
protocol = "http"
destination = "backend.internal"
source = "*"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{"check", "--config", path}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	if !strings.Contains(stdout.String(), `"name": "public"`) {
		t.Fatalf("stdout = %q, want canonical config", stdout.String())
	}
}

func TestRunExplainPrintsDecisionWithoutServing(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "delegate.toml")
	contents := []byte(`
[[servers]]
name = "public"
protocol = "http"
listen = ":8080"

[[mounts]]
path = "/api/*"
target = "http://api.internal/v1/*"

[[policies]]
effect = "permit"
protocol = "http"
destination = "api.internal"
source = "10.0.0.0/8"
method = "GET"
mount = "/api/*"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{
		"explain", "--config", path, "--path", "/api/users",
		"--source", "10.20.30.40", "--method", "GET",
	}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	for _, fragment := range []string{`"outcome": "permit"`, `"target": "http://api.internal/v1/users"`, `"rule_index": 0`} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout missing %q:\n%s", fragment, stdout.String())
		}
	}
}

func TestRunExplainRejectsMissingRequestInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"explain", "SERVER=http", "--path", "/"}, &stdout, &stderr, func(config.Config) error {
		t.Fatal("serve called by explain")
		return nil
	})
	if exitCode != 2 || !strings.Contains(stderr.String(), "--source") {
		t.Fatalf("run() = %d, stderr=%q; want missing source error", exitCode, stderr.String())
	}
}

func TestRunExplainAcceptsLegacyDirectives(t *testing.T) {
	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{
		"explain",
		"SERVER=http", "-P8080",
		`MOUNT="/* http://backend.internal/*"`,
		`PERMIT="http:backend.internal:*"`,
		"--path", "/docs", "--source", "192.0.2.4", "--method", "GET",
	}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})
	if exitCode != 0 {
		t.Fatalf("run() = %d, stderr=%q; want success", exitCode, stderr.String())
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	if !strings.Contains(stdout.String(), `"outcome": "permit"`) {
		t.Fatalf("stdout = %q, want permit explanation", stdout.String())
	}
}

func TestRunRejectsConfigFileMixedWithLegacyDirectives(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"check", "--config", "delegate.toml", "SERVER=http"}, &stdout, &stderr, func(config.Config) error {
		t.Fatal("serve called for invalid arguments")
		return nil
	})
	if exitCode != 2 || !strings.Contains(stderr.String(), "cannot combine") {
		t.Fatalf("run() = %d, stderr=%q; want cannot combine error", exitCode, stderr.String())
	}
}

func TestRunReportsConfigFileReadFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"check", "--config", filepath.Join(t.TempDir(), "missing.toml")}, &stdout, &stderr, func(config.Config) error {
		t.Fatal("serve called for unreadable configuration")
		return nil
	})
	if exitCode != 2 || !strings.Contains(stderr.String(), "read configuration") {
		t.Fatalf("run() = %d, stderr=%q; want read error", exitCode, stderr.String())
	}
}

func TestRunRejectsMissingConfigPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"check", "--config"}, &stdout, &stderr, func(config.Config) error {
		t.Fatal("serve called for missing configuration path")
		return nil
	})
	if exitCode != 2 || !strings.Contains(stderr.String(), "requires a path") {
		t.Fatalf("run() = %d, stderr=%q; want path error", exitCode, stderr.String())
	}
}

func TestRunRejectsInvalidConfigurationBeforeServing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{"SERVER=http", "-Pnope"}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})

	if exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2", exitCode)
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	if !strings.Contains(stderr.String(), "invalid configuration") {
		t.Fatalf("stderr = %q, want validation error", stderr.String())
	}
}

func TestRunRejectsUnsupportedFrontendBeforeServing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	serveCalls := 0
	exitCode := run([]string{"SERVER=ftp"}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})

	if exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2", exitCode)
	}
	if serveCalls != 0 {
		t.Fatalf("serve calls = %d, want zero", serveCalls)
	}
	if !strings.Contains(stderr.String(), "unsupported frontend protocol") {
		t.Fatalf("stderr = %q, want unsupported protocol error", stderr.String())
	}
}

func TestRunServesValidatedDirectives(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var served config.Config
	exitCode := run([]string{
		"SERVER=http",
		"-P8080",
		`MOUNT="/* http://backend.internal/*"`,
		`PERMIT="http:backend.internal:*"`,
	}, &stdout, &stderr, func(got config.Config) error {
		served = got
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if len(served.Servers) != 1 || served.Servers[0].Listen != ":8080" {
		t.Fatalf("served config = %#v, want validated :8080 server", served)
	}
}

func TestRunServesMultipleValidatedHTTPListeners(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "delegate.toml")
	contents := []byte(`
[[servers]]
name = "public"
protocol = "http"
listen = "127.0.0.1:8080"

[[servers]]
name = "admin"
protocol = "http"
listen = "127.0.0.1:8081"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	var served config.Config
	exitCode := run([]string{"serve", "--config", path}, &stdout, &stderr, func(got config.Config) error {
		served = got
		return nil
	})
	if exitCode != 0 {
		t.Fatalf("run() = %d, stderr=%q; want success", exitCode, stderr.String())
	}
	if len(served.Servers) != 2 {
		t.Fatalf("served %d servers, want 2", len(served.Servers))
	}
}

func TestRunChecksAndPassesTLSConfigToRuntime(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "delegate.toml")
	contents := []byte(`
[[servers]]
name = "public"
protocol = "http"
listen = "127.0.0.1:8080"
[servers.tls]
certificate_file = "certs/frontend.pem"
private_key_file = "certs/frontend-key.pem"
minimum_version = "1.3"

[[mounts]]
path = "/*"
target = "https://backend.internal/*"
[mounts.tls]
ca_file = "certs/backend-ca.pem"
minimum_version = "1.2"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	var checkOut, checkErr bytes.Buffer
	if got := run([]string{"check", "--config", path}, &checkOut, &checkErr, func(config.Config) error {
		t.Fatal("serve called by check")
		return nil
	}); got != 0 {
		t.Fatalf("check = %d, stderr=%q", got, checkErr.String())
	}
	if !strings.Contains(checkOut.String(), `"minimum_version": "1.3"`) {
		t.Fatalf("check output does not contain frontend TLS: %s", checkOut.String())
	}

	var stdout, stderr bytes.Buffer
	serveCalls := 0
	got := run([]string{"serve", "--config", path}, &stdout, &stderr, func(config.Config) error {
		serveCalls++
		return nil
	})
	if got != 0 || serveCalls != 1 {
		t.Fatalf("serve = %d, calls=%d, stderr=%q; want runtime handoff", got, serveCalls, stderr.String())
	}
}

func TestRuntimeSupportAcceptsBackendTLSIndependently(t *testing.T) {
	configured := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
		Mounts: []mount.Mount{{
			Path: "/*", Target: "https://backend.internal/*",
			TLS: &tlsconfig.Backend{CAFile: "certs/backend-ca.pem"},
		}},
	}
	err := validateRuntimeSupport(configured)
	if err != nil {
		t.Fatalf("validateRuntimeSupport() error = %v", err)
	}
}

func TestRunPassesFrontendTLSToRuntime(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "delegate.toml")
	contents := []byte(`
[[servers]]
name = "secure"
protocol = "http"
listen = "127.0.0.1:8443"
[servers.tls]
certificate_file = "certs/frontend.pem"
private_key_file = "certs/frontend-key.pem"
minimum_version = "1.3"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	serveCalls := 0
	got := run([]string{"serve", "--config", path}, &stdout, &stderr, func(configured config.Config) error {
		serveCalls++
		if len(configured.Servers) != 1 || configured.Servers[0].TLS == nil {
			t.Fatalf("runtime config = %#v, want frontend TLS", configured)
		}
		return nil
	})
	if got != 0 || serveCalls != 1 {
		t.Fatalf("run() = %d, calls=%d, stderr=%q", got, serveCalls, stderr.String())
	}
}

func TestRunReportsServeFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"SERVER=http"}, &stdout, &stderr, func(config.Config) error {
		return errors.New("listen failed")
	})
	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "listen failed") {
		t.Fatalf("stderr = %q, want serve error", stderr.String())
	}
}

func TestConfigPathFromArgs(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"serve", "--config", "delegate.toml"}, want: "delegate.toml"},
		{args: []string{"--config=delegate.toml"}, want: "delegate.toml"},
		{args: []string{"SERVER=http"}},
	}
	for _, tt := range tests {
		if got := configPathFromArgs(tt.args); got != tt.want {
			t.Fatalf("configPathFromArgs(%q) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestWatchReloadPublishesChangesAndReportsFailures(t *testing.T) {
	initial := config.Config{
		Servers: []config.Server{{Name: "public", Protocol: "http", Listen: ":8080"}},
	}
	store, err := config.NewStore(initial)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delegate.toml")
	write := func(extra string) {
		t.Helper()
		text := `[[servers]]
name = "public"
protocol = "http"
listen = ":8080"
` + extra
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("\n[[mounts]]\npath = \"/*\"\ntarget = \"http://new.internal/*\"\n")

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan os.Signal, 1)
	reports := make(chan error, 3)
	done := make(chan struct{})
	go func() {
		watchReload(ctx, events, store, path, func(err error) { reports <- err })
		close(done)
	}()
	events <- fakeSignal("reload")
	if err := <-reports; err != nil {
		t.Fatalf("valid reload report = %v", err)
	}
	if got := store.Snapshot().Mounts[0].Target; got != "http://new.internal/*" {
		t.Fatalf("target = %q, want reloaded target", got)
	}

	write("\nunknown = true\n")
	events <- fakeSignal("reload")
	if err := <-reports; err == nil {
		t.Fatal("invalid reload report = nil")
	}
	if got := store.Snapshot().Mounts[0].Target; got != "http://new.internal/*" {
		t.Fatalf("invalid reload changed target to %q", got)
	}

	write(`[servers.tls]
certificate_file = "certs/frontend.pem"
private_key_file = "certs/frontend-key.pem"
`)
	events <- fakeSignal("reload")
	if err := <-reports; err == nil || !strings.Contains(err.Error(), "listener topology") {
		t.Fatalf("unsupported TLS reload report = %v", err)
	}
	if got := store.Snapshot().Servers[0].TLS; got != nil {
		t.Fatalf("unsupported reload published frontend TLS: %#v", got)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reload watcher did not stop")
	}
}

type fakeSignal string

func (s fakeSignal) String() string { return string(s) }
func (fakeSignal) Signal()          {}
