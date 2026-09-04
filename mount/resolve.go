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

// Request identifies both the resource path and the frontend evaluating it.
type Request struct {
	Path      string
	Scheme    string
	Authority string
	Server    string
	Protocol  string
}

// Resolve safely normalizes a request path and selects the most specific
// mapping. Priority only breaks ties between equally specific patterns.
func Resolve(mounts []Mount, requestPath string) (Match, error) {
	return ResolveFor(mounts, Request{Path: requestPath})
}

// ResolveFor resolves a normalized path and optional URL authority after
// filtering mappings by named server and frontend protocol. Empty and "*"
// scopes match every frontend.
func ResolveFor(mounts []Mount, request Request) (Match, error) {
	normalized, err := NormalizePath(request.Path)
	if err != nil {
		return Match{}, fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}

	type candidate struct {
		mapping     Mount
		capture     string
		specificity int
		scope       int
		authority   int
	}
	var winner candidate
	found := false
	ambiguous := false

	for _, mapping := range mounts {
		scope, ok := matchScope(mapping, request)
		if !ok {
			continue
		}
		pattern := mapping.Path
		authority := 0
		if mapping.Source != "" {
			parsed, parseErr := parseSourceURL(mapping.Source)
			if parseErr != nil || !strings.EqualFold(parsed.Scheme, request.Scheme) || !strings.EqualFold(parsed.Host, request.Authority) {
				continue
			}
			pattern = parsed.EscapedPath()
			if pattern == "" {
				pattern = "/"
			}
			authority = 1
		}
		capture, specificity, ok := matchPath(pattern, normalized)
		if !ok {
			continue
		}
		current := candidate{mapping: mapping, capture: capture, specificity: specificity, scope: scope, authority: authority}
		if !found || current.specificity > winner.specificity ||
			(current.specificity == winner.specificity && current.mapping.Priority > winner.mapping.Priority) ||
			(current.specificity == winner.specificity && current.mapping.Priority == winner.mapping.Priority && current.authority > winner.authority) ||
			(current.specificity == winner.specificity && current.mapping.Priority == winner.mapping.Priority && current.authority == winner.authority && current.scope > winner.scope) {
			winner = current
			found = true
			ambiguous = false
			continue
		}
		if current.specificity == winner.specificity && current.mapping.Priority == winner.mapping.Priority && current.authority == winner.authority && current.scope == winner.scope {
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

func matchScope(mapping Mount, request Request) (int, bool) {
	specificity := 0
	if mapping.Server != "" && mapping.Server != "*" {
		if mapping.Server != request.Server {
			return 0, false
		}
		specificity++
	}
	if mapping.Protocol != "" && mapping.Protocol != "*" {
		if !strings.EqualFold(mapping.Protocol, request.Protocol) {
			return 0, false
		}
		specificity++
	}
	return specificity, true
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
