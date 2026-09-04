package compatibility

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFixtureSuiteLoadsNestedFixtureFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fixture := `{"name":"SERVER=http with -P8080","args":["SERVER=http","-P8080"],"reference_config":{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}}`

	if err := os.WriteFile(filepath.Join(tempDir, "root.json"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(tempDir, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.json"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	nestedIgnore := filepath.Join(tempDir, "nested", "ignore.md")
	if err := os.WriteFile(nestedIgnore, []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}

	mismatches, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err != nil {
		t.Fatalf("RunFixtureSuite() = %v", err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("RunFixtureSuite() mismatches = %d, want 0", len(mismatches))
	}

	names, err := listFixtureFiles(tempDir)
	if err != nil {
		t.Fatalf("listFixtureFiles() = %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("listFixtureFiles() = %v, want 2", names)
	}
	if names[0] != "nested/deep.json" || names[1] != "root.json" {
		t.Fatalf("listFixtureFiles() order = %v, want [nested/deep.json root.json]", names)
	}
}

func TestRunFixtureSuiteReturnsErrorWhenNoFixturesExist(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	_, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err == nil {
		t.Fatal("RunFixtureSuite() = nil, want error")
	}
	if !strings.Contains(err.Error(), "no fixture files found") {
		t.Fatalf("RunFixtureSuite() error = %v, want contains %q", err, "no fixture files found")
	}
}

func TestListFixtureFilesReturnsErrorForMissingDirectory(t *testing.T) {
	t.Parallel()

	if _, err := listFixtureFiles(filepath.Join("testdata", "does-not-exist")); err == nil {
		t.Fatal("listFixtureFiles() = nil, want error")
	}
}

func TestListFixtureFilesReturnsErrorForNonDirectoryPath(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), "not-a-dir.json")
	if err := os.WriteFile(tempFile, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := listFixtureFiles(tempFile); err == nil {
		t.Fatal("listFixtureFiles() = nil, want error")
	}
}

func TestRunFixtureSuitePropagatesInvalidFixtureError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "bad.json"), []byte(`{
  "name": "bad",
  "reference_config": {"servers":[{"name":"default","protocol":"http","listen":":8080"}]}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err == nil {
		t.Fatal("RunFixtureSuite() = nil, want error")
	}
	if !strings.Contains(err.Error(), "fixture \"bad.json\"") || !strings.Contains(err.Error(), "missing args") {
		t.Fatalf("RunFixtureSuite() error = %v, want %q", err, "fixture \"bad.json\": ... missing args")
	}
}

func TestRunFixtureSuiteRejectsCancelledContextBeforeReferenceExecution(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fixture := `{"name":"SERVER=http with -P8080","args":["SERVER=http","-P8080"],"reference_config":{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}}`
	if err := os.WriteFile(filepath.Join(tempDir, "fixture.json"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runnerCalled := false
	_, err := RunFixtureSuite(ctx, tempDir, CompareOptions{
		Runner: func(context.Context, ...string) ([]byte, error) {
			runnerCalled = true
			return []byte(`{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}`), nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunFixtureSuite() error = %v, want context.Canceled", err)
	}
	if runnerCalled {
		t.Fatal("RunFixtureSuite() called reference runner after cancellation")
	}
}

func TestRunFixtureSuiteStopsBetweenFixturesWhenContextIsCancelled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	fixture := `{"name":"SERVER=http with -P8080","args":["SERVER=http","-P8080"],"reference_config":{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}}`
	for _, name := range []string{"a.json", "b.json"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(fixture), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runnerCalls := 0
	_, err := RunFixtureSuite(ctx, tempDir, CompareOptions{
		Runner: func(context.Context, ...string) ([]byte, error) {
			runnerCalls++
			cancel()
			return []byte(`{"servers":[{"name":"default","protocol":"http","listen":":8080"}]}`), nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunFixtureSuite() error = %v, want context.Canceled", err)
	}
	if runnerCalls != 1 {
		t.Fatalf("reference runner calls = %d, want 1", runnerCalls)
	}
}
