package compatibility

import (
	"encoding/json"
	"fmt"
	"os"

	"gitea.local/ryan/new-delegate/config"
)

// Fixture captures one compatibility checkpoint where legacy directives should
// produce a known canonical configuration.
type Fixture struct {
	Name            string        `json:"name"`
	Args            []string      `json:"args"`
	ReferenceConfig config.Config `json:"reference_config"`
}

// LoadFixture reads and validates a compatibility fixture.
func LoadFixture(path string) (Fixture, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, fmt.Errorf("read fixture %q: %w", path, err)
	}

	var fixture Fixture
	if err := json.Unmarshal(encoded, &fixture); err != nil {
		return Fixture{}, fmt.Errorf("decode fixture %q: %w", path, err)
	}
	if fixture.Name == "" {
		return Fixture{}, fmt.Errorf("fixture %q: missing name", path)
	}
	if len(fixture.Args) == 0 {
		return Fixture{}, fmt.Errorf("fixture %q: missing args", path)
	}
	if err := fixture.ReferenceConfig.Validate(); err != nil {
		return Fixture{}, fmt.Errorf("fixture %q: reference_config invalid: %w", path, err)
	}
	return fixture, nil
}
