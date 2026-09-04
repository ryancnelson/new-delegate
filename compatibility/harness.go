package compatibility

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"

	"gitea.local/ryan/new-delegate/config"
)

// ReferenceRunner executes the reference implementation for the same legacy
// directives.
type ReferenceRunner func(ctx context.Context, args ...string) ([]byte, error)

// CompareOptions controls how compatibility checks compare new behavior against
// prior output.
type CompareOptions struct {
	Runner  ReferenceRunner
	BinPath string
}

// FixtureMismatch captures structured output when canonical models diverge.
type FixtureMismatch struct {
	Name     string
	Expected config.Config
	Actual   config.Config
}

func (m FixtureMismatch) Error() string {
	return "compatibility mismatch for " + m.Name
}

// CompareFixture compares the parsed fixture result from current code with either
// a configured reference binary output or a fixed reference config in the
// fixture.
func CompareFixture(fixture Fixture, opts CompareOptions) (FixtureMismatch, error) {
	actual, err := config.ParseLegacyArgs(fixture.Args)
	if err != nil {
		return FixtureMismatch{}, fmt.Errorf("new parse failure for %q: %w", fixture.Name, err)
	}

	expected := fixture.ReferenceConfig
	if !isReferenceConfigured(opts) {
		return FixtureMismatch{}, compare(fixture.Name, actual, expected)
	}

	var reference []byte
	reference, err = runReferenceRunner(context.Background(), fixture.Args, opts)
	if err != nil {
		return FixtureMismatch{}, fmt.Errorf("reference comparison for %q: %w", fixture.Name, err)
	}

	var decoded config.Config
	if err := json.Unmarshal(reference, &decoded); err != nil {
		return FixtureMismatch{}, fmt.Errorf("decode reference output for %q: %w", fixture.Name, err)
	}
	if err := decoded.Validate(); err != nil {
		return FixtureMismatch{}, fmt.Errorf("reference output invalid for %q: %w", fixture.Name, err)
	}
	expected = decoded
	return FixtureMismatch{}, compare(fixture.Name, actual, expected)
}

func compare(name string, actual, expected config.Config) error {
	if err := actual.Validate(); err != nil {
		return fmt.Errorf("actual parse for %q invalid: %w", name, err)
	}
	if reflect.DeepEqual(actual, expected) {
		return nil
	}
	return FixtureMismatch{Name: name, Expected: expected, Actual: actual}
}

func isReferenceConfigured(options CompareOptions) bool {
	return options.Runner != nil || options.BinPath != ""
}

func runReferenceRunner(ctx context.Context, args []string, options CompareOptions) ([]byte, error) {
	if options.Runner != nil {
		return options.Runner(ctx, args...)
	}
	command := exec.CommandContext(ctx, options.BinPath, append([]string{"check"}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", options.BinPath, err)
	}
	return output, nil
}
