package compatibility

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunFixtureSuite runs all compatibility fixtures in a directory and returns
// every structured mismatch discovered.
//
// Fixtures are processed in lexical filename order to keep results deterministic.
func RunFixtureSuite(ctx context.Context, fixtureDir string, options CompareOptions) ([]FixtureMismatch, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	options.Context = ctx

	names, err := listFixtureFiles(fixtureDir)
	if err != nil {
		return nil, fmt.Errorf("read fixture directory %q: %w", fixtureDir, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no fixture files found in %q", fixtureDir)
	}

	var mismatches []FixtureMismatch
	for _, name := range names {
		path := filepath.Join(fixtureDir, name)
		fixture, err := LoadFixture(path)
		if err != nil {
			return nil, fmt.Errorf("fixture %q: %w", name, err)
		}

		_, err = CompareFixture(fixture, options)
		if err == nil {
			continue
		}

		var fixtureMismatch FixtureMismatch
		if errors.As(err, &fixtureMismatch) {
			mismatches = append(mismatches, fixtureMismatch)
			continue
		}

		return nil, fmt.Errorf("compare fixture %q: %w", name, err)
	}

	return mismatches, nil
}

func listFixtureFiles(base string) ([]string, error) {
	info, err := os.Stat(base)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", base)
	}

	var names []string
	if err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(names)
	return names, nil
}
