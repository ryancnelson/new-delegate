package mount

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"
)

// NormalizePath decodes and canonicalizes a request path before matching.
func NormalizePath(raw string) (string, error) {
	if !strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("path must be absolute")
	}
	if hasEncodedSeparator(raw) {
		return "", fmt.Errorf("encoded path separator is not allowed")
	}

	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", fmt.Errorf("invalid path escape: %w", err)
	}
	if !utf8.ValidString(decoded) {
		return "", fmt.Errorf("path is not valid UTF-8")
	}
	if strings.Contains(decoded, "\\") {
		return "", fmt.Errorf("backslash is not allowed in a path")
	}
	for _, r := range decoded {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("control character is not allowed in a path")
		}
	}
	if containsPercentEscape(decoded) {
		return "", fmt.Errorf("possible double encoding is not allowed")
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("path traversal segment %q is not allowed", segment)
		}
	}

	trailingSlash := strings.HasSuffix(decoded, "/")
	normalized := path.Clean(decoded)
	if trailingSlash && normalized != "/" {
		normalized += "/"
	}
	return normalized, nil
}

func hasEncodedSeparator(value string) bool {
	for i := 0; i+2 < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		decoded, ok := decodeHexByte(value[i+1], value[i+2])
		if ok && (decoded == '/' || decoded == '\\') {
			return true
		}
		i += 2
	}
	return false
}

func containsPercentEscape(value string) bool {
	for i := 0; i+2 < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		if _, ok := decodeHexByte(value[i+1], value[i+2]); ok {
			return true
		}
	}
	return false
}

func decodeHexByte(high, low byte) (byte, bool) {
	hi, ok := hexNibble(high)
	if !ok {
		return 0, false
	}
	lo, ok := hexNibble(low)
	if !ok {
		return 0, false
	}
	return hi<<4 | lo, true
}

func hexNibble(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}
