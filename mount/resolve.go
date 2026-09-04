package mount

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNoMatch    = errors.New("no matching mount")
	ErrAmbiguous  = errors.New("ambiguous mount")
	ErrUnsafePath = errors.New("unsafe request path")
)

// Match is the selected mapping and its rewritten backend target.
type Match struct {
	Mount          Mount
	NormalizedPath string
	Target         string
}

// Resolve safely normalizes a request path and selects the most specific
// mapping. Priority only breaks ties between equally specific patterns.
func Resolve(mounts []Mount, requestPath string) (Match, error) {
	normalized, err := NormalizePath(requestPath)
	if err != nil {
		return Match{}, fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}

	type candidate struct {
		mapping     Mount
		capture     string
		specificity int
	}
	var winner candidate
	found := false
	ambiguous := false

	for _, mapping := range mounts {
		capture, specificity, ok := matchPath(mapping.Path, normalized)
		if !ok {
			continue
		}
		current := candidate{mapping: mapping, capture: capture, specificity: specificity}
		if !found || current.specificity > winner.specificity ||
			(current.specificity == winner.specificity && current.mapping.Priority > winner.mapping.Priority) {
			winner = current
			found = true
			ambiguous = false
			continue
		}
		if current.specificity == winner.specificity && current.mapping.Priority == winner.mapping.Priority {
			ambiguous = true
		}
	}

	if !found {
		return Match{}, ErrNoMatch
	}
	if ambiguous {
		return Match{}, ErrAmbiguous
	}
	target := winner.mapping.Target
	if strings.HasSuffix(target, "*") {
		target = strings.TrimSuffix(target, "*") + winner.capture
	}
	return Match{Mount: winner.mapping, NormalizedPath: normalized, Target: target}, nil
}

func matchPath(pattern, requestPath string) (capture string, specificity int, ok bool) {
	if !strings.HasSuffix(pattern, "*") {
		if pattern == requestPath {
			return "", len(pattern) + 1, true
		}
		return "", 0, false
	}
	prefix := strings.TrimSuffix(pattern, "*")
	if !strings.HasPrefix(requestPath, prefix) {
		return "", 0, false
	}
	return strings.TrimPrefix(requestPath, prefix), len(prefix), true
}
