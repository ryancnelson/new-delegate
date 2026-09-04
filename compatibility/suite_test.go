package compatibility

import (
	"context"
	"os"
	"path/filepath"
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

	mismatches, err := RunFixtureSuite(context.Background(), tempDir, CompareOptions{})
	if err != nil {
		t.Fatalf("RunFixtureSuite() = %v", err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("RunFixtureSuite() mismatches = %d, want 0", len(mismatches))
	}
}
