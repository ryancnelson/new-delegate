package config

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ParseTOML decodes and validates canonical configuration. Reading files is
// intentionally left to callers so decoding remains free of runtime side
// effects and is straightforward to test.
func ParseTOML(input io.Reader) (Config, error) {
	if input == nil {
		return Config{}, fmt.Errorf("decode TOML: input is required")
	}

	var result Config
	decoder := toml.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		var unknown *toml.StrictMissingError
		if errors.As(err, &unknown) {
			keys := make([]string, 0, len(unknown.Errors))
			for i := range unknown.Errors {
				keys = append(keys, strings.Join(unknown.Errors[i].Key(), "."))
			}
			return Config{}, fmt.Errorf("decode TOML: unknown field(s): %s", strings.Join(keys, ", "))
		}
		var syntax *toml.DecodeError
		if errors.As(err, &syntax) {
			row, column := syntax.Position()
			return Config{}, fmt.Errorf("decode TOML at line %d, column %d: %s", row, column, syntax.Error())
		}
		return Config{}, fmt.Errorf("decode TOML: %w", err)
	}
	if err := result.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate TOML: %w", err)
	}
	return result, nil
}
