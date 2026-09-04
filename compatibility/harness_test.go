package compatibility

import (
	"context"
	"encoding/json"
	"errors"
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

	_, err = LoadFixture(filepath.Join("testdata", "fixture-http-mount-permit.json"))
	if err != nil {
		t.Fatalf("LoadFixture() = %v", err)
	}

	_, err = LoadFixture(filepath.Join("testdata", "fixture-http-mount-scoped.json"))
	if err != nil {
		t.Fatalf("LoadFixture() = %v", err)
	}

	_, err = LoadFixture(filepath.Join("testdata", "fixture-http-connect-mount.json"))
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

func TestRunFixtureSuite(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join("testdata", "fixture-server-8080.json")
	dst := filepath.Join(tempDir, "fixture-server-8080.json")

	encoded, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	mismatches, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err != nil {
		t.Fatalf("RunFixtureSuite() = %v, want nil", err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("RunFixtureSuite() mismatches = %d, want 0", len(mismatches))
	}
}

func TestRunFixtureSuiteReportsMismatch(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join("testdata", "fixture-server-8080.json")
	dst := filepath.Join(tempDir, "fixture-server-8080.json")

	encoded, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var fixture Fixture
	if err := json.Unmarshal(encoded, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.ReferenceConfig.Servers[0].Listen = ":9090"
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, mutated, 0o600); err != nil {
		t.Fatal(err)
	}

	mismatches, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err != nil {
		t.Fatalf("RunFixtureSuite() = %v, want nil", err)
	}
	if len(mismatches) != 1 {
		t.Fatalf("RunFixtureSuite() mismatches = %d, want 1", len(mismatches))
	}
	if mismatches[0].Name != "SERVER=http with -P8080" {
		t.Fatalf("mismatch name = %q, want %q", mismatches[0].Name, "SERVER=http with -P8080")
	}
}

func TestRunFixtureSuiteAggregatesMultipleResults(t *testing.T) {
	tempDir := t.TempDir()

	rawGood, err := os.ReadFile(filepath.Join("testdata", "fixture-server-8080.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "a-base.json"), rawGood, 0o600); err != nil {
		t.Fatal(err)
	}

	var mismatch Fixture
	if err := json.Unmarshal(rawGood, &mismatch); err != nil {
		t.Fatal(err)
	}
	mismatch.Name = "A mismatched fixture"
	mismatch.ReferenceConfig.Servers[0].Listen = ":9090"
	mutated, err := json.Marshal(mismatch)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "z-bad.json"), mutated, 0o600); err != nil {
		t.Fatal(err)
	}

	mismatches, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err != nil {
		t.Fatalf("RunFixtureSuite() = %v, want nil", err)
	}
	if len(mismatches) != 1 {
		t.Fatalf("RunFixtureSuite() mismatches = %d, want 1", len(mismatches))
	}
	if mismatches[0].Name != "A mismatched fixture" {
		t.Fatalf("mismatch name = %q, want %q", mismatches[0].Name, "A mismatched fixture")
	}
}

func TestRunFixtureSuitePropagatesRunnerError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "fixture.json"), []byte(`{
  "name": "bad ref",
  "args": ["SERVER=http", "-P8080"],
  "reference_config": {
    "servers": [
      {"name":"default","protocol":"http","listen":":8080"}
    ]
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runnerErr := errors.New("reference unavailable")
	runner := func(_ context.Context, _ ...string) ([]byte, error) { return nil, runnerErr }

	_, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{Runner: runner})
	if err == nil {
		t.Fatal("RunFixtureSuite() = nil, want error")
	}
	if !strings.Contains(err.Error(), "reference unavailable") {
		t.Fatalf("RunFixtureSuite() error = %v, want contains %q", err, "reference unavailable")
	}
}
