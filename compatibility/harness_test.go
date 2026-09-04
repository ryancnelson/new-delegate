package compatibility

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadFixture(t *testing.T) {
	_, err := LoadFixture(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatalf("LoadFixture() = %v", err)
	}
}

func TestLoadFixtureRejectsEmptyArgs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"name":"bad","args":[],"reference_config":{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(path); err == nil {
		t.Fatal("LoadFixture() = nil, want error")
	} else if !strings.Contains(err.Error(), "missing args") {
		t.Fatalf("LoadFixture() error = %v, want contains %q", err, "missing args")
	}
}

func TestCompareFixtureAgainstStoredReference(t *testing.T) {
	fixture, err := LoadFixture(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CompareFixture(fixture, CompareOptions{}); err != nil {
		t.Fatalf("CompareFixture() = %v, want nil", err)
	}
}

func TestCompareFixtureDetectsMismatch(t *testing.T) {
	fixture, err := LoadFixture(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture.ReferenceConfig.Servers[0].Listen = ":9090"

	_, err = CompareFixture(fixture, CompareOptions{})
	if err == nil {
		t.Fatal("CompareFixture() = nil, want mismatch")
	}
	if !strings.Contains(err.Error(), "compatibility mismatch for SERVER=http with -P8080") {
		t.Fatalf("CompareFixture() error = %v, want mismatch", err)
	}
}

func TestCompareFixtureWithReferenceRunner(t *testing.T) {
	fixture, err := LoadFixture(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := json.Marshal(fixture.ReferenceConfig)
	if err != nil {
		t.Fatal(err)
	}

	runner := func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 0 || !strings.Contains(args[0], "SERVER") {
			t.Fatalf("unexpected args %v", args)
		}
		return append(append([]byte{}, baseline...), '\n'), nil
	}

	if _, err := CompareFixture(fixture, CompareOptions{Runner: runner}); err != nil {
		t.Fatalf("CompareFixture() = %v, want nil", err)
	}
}

func TestRunReferenceRunnerFallsBackToBinPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("binary runner test uses shell scripts")
	}

	temp := t.TempDir()
	script := filepath.Join(temp, "delegate")
	scriptText := "#!/usr/bin/env sh\nprintf '%s\\n' '{\"servers\":[{\"name\":\"default\",\"protocol\":\"http\",\"listen\":\":8080\"}]}'\n"
	if err := os.WriteFile(script, []byte(scriptText), 0o700); err != nil {
		t.Fatal(err)
	}

	fixture, err := LoadFixture(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = CompareFixture(fixture, CompareOptions{BinPath: script})
	if err != nil {
		t.Fatalf("CompareFixture() = %v, want nil", err)
	}
}
