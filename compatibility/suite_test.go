package compatibility

import (
	"context"
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
