package main

import (
	"bytes"
	"errors"
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
