// Package release builds deterministic, checksummed distribution archives.
package release

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Target is one supported release platform.
type Target struct {
	GOOS   string
	GOARCH string
}

// Targets is the stable release order and supported portability matrix.
var Targets = []Target{
	{GOOS: "darwin", GOARCH: "arm64"},
	{GOOS: "linux", GOARCH: "amd64"},
}

// Builder creates one target executable at output.
type Builder func(ctx context.Context, target Target, output, version string) error

var safeVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Build creates one deterministic zip per target and a SHA-256 manifest.
func Build(ctx context.Context, sourceRoot, outputDirectory, version string, builder Builder) error {
	if !safeVersion.MatchString(version) {
		return fmt.Errorf("invalid release version %q", version)
	}
	if builder == nil {
		return fmt.Errorf("release builder is required")
	}
	readme, err := os.ReadFile(filepath.Join(sourceRoot, "README.md"))
	if err != nil {
		return fmt.Errorf("read release README: %w", err)
	}
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf("create release directory: %w", err)
	}
	staging, err := os.MkdirTemp("", "new-delegate-release-")
	if err != nil {
		return fmt.Errorf("create release staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	manifest := make([]byte, 0, len(Targets)*128)
	for _, target := range Targets {
		executableName := "delegate"
		if target.GOOS == "windows" {
			executableName += ".exe"
		}
		executablePath := filepath.Join(staging, target.GOOS+"-"+target.GOARCH+"-"+executableName)
		if err := builder(ctx, target, executablePath, version); err != nil {
			return fmt.Errorf("build %s/%s: %w", target.GOOS, target.GOARCH, err)
		}
		executable, err := os.ReadFile(executablePath)
		if err != nil {
			return fmt.Errorf("read %s/%s executable: %w", target.GOOS, target.GOARCH, err)
		}
		archiveName := ArchiveName(version, target)
		archivePath := filepath.Join(outputDirectory, archiveName)
		if err := writeArchive(archivePath, executableName, executable, readme); err != nil {
			return err
		}
		archive, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("read release archive %q: %w", archiveName, err)
		}
		sum := sha256.Sum256(archive)
		manifest = fmt.Appendf(manifest, "%x  %s\n", sum, archiveName)
	}
	if err := os.WriteFile(filepath.Join(outputDirectory, "SHA256SUMS"), manifest, 0o644); err != nil {
		return fmt.Errorf("write release checksums: %w", err)
	}
	return nil
}

// ArchiveName returns the stable artifact filename for one target.
func ArchiveName(version string, target Target) string {
	return fmt.Sprintf("new-delegate-%s-%s-%s.zip", version, target.GOOS, target.GOARCH)
}

func writeArchive(destination, executableName string, executable, readme []byte) error {
	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create release archive: %w", err)
	}
	written := false
	defer func() {
		if !written {
			_ = file.Close()
		}
	}()
	archive := zip.NewWriter(file)
	fixedTime := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, entry := range []struct {
		name string
		mode os.FileMode
		data []byte
	}{
		{name: executableName, mode: 0o755, data: executable},
		{name: "README.md", mode: 0o644, data: readme},
	} {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate, Modified: fixedTime}
		header.SetMode(entry.mode)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create archive entry %q: %w", entry.name, err)
		}
		if _, err := writer.Write(entry.data); err != nil {
			return fmt.Errorf("write archive entry %q: %w", entry.name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("finalize release archive: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close release archive: %w", err)
	}
	written = true
	return nil
}
