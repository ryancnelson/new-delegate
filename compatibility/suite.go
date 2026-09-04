package compatibility

import (
	"context"
	"errors"
	"fmt"
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

	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		return nil, fmt.Errorf("read fixture directory %q: %w", fixtureDir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
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
