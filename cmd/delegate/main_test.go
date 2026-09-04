package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.local/ryan/new-delegate/config"
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
