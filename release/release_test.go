package release

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTargetsMatchActiveBuildMatrix(t *testing.T) {
	want := []Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
	}
	if !slices.Equal(Targets, want) {
		t.Fatalf("Targets = %#v, want %#v", Targets, want)
	}
}

func TestBuildCreatesDeterministicChecksummedArchives(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("release notice\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	builder := func(_ context.Context, target Target, output, version string) error {
		return os.WriteFile(output, []byte(version+" "+target.GOOS+"/"+target.GOARCH+"\n"), 0o755)
	}
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	if err := Build(context.Background(), source, first, "v1.2.3", builder); err != nil {
		t.Fatal(err)
	}
	if err := Build(context.Background(), source, second, "v1.2.3", builder); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(Targets)+1 {
		t.Fatalf("release files = %d, want %d archives plus checksums", len(entries), len(Targets))
	}
	checksums, err := os.ReadFile(filepath.Join(first, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(checksums)), "\n")
	if len(lines) != len(Targets) {
		t.Fatalf("checksum lines = %d, want %d", len(lines), len(Targets))
	}

	for _, target := range Targets {
		name := ArchiveName("v1.2.3", target)
		firstBytes, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		secondBytes, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(firstBytes, secondBytes) {
			t.Fatalf("archive %s is not deterministic", name)
		}
		archive, err := zip.OpenReader(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(archive.File) != 2 || archive.File[1].Name != "README.md" {
			_ = archive.Close()
			t.Fatalf("archive %s entries = %#v", name, archive.File)
		}
		wantExecutable := "delegate"
		if target.GOOS == "windows" {
			wantExecutable += ".exe"
		}
		if archive.File[0].Name != wantExecutable || archive.File[0].Mode().Perm() != 0o755 {
			_ = archive.Close()
			t.Fatalf("archive executable = %s mode %o", archive.File[0].Name, archive.File[0].Mode().Perm())
		}
		_ = archive.Close()
		if !strings.Contains(string(checksums), "  "+name) {
			t.Fatalf("SHA256SUMS missing %s", name)
		}
	}
}

func TestBuildRejectsUnsafeVersion(t *testing.T) {
	err := Build(context.Background(), t.TempDir(), t.TempDir(), "../latest", func(context.Context, Target, string, string) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("Build() error = %v, want version rejection", err)
	}
}
