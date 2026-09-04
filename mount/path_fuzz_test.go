package mount

import "testing"

func FuzzNormalizePath(f *testing.F) {
	for _, seed := range []string{"/", "/a/b", "/a//b/", "/%2e%2e/x", "/%00", "relative"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		normalized, err := NormalizePath(raw)
		if err != nil {
			return
		}
		again, err := NormalizePath(normalized)
		if err != nil {
			t.Fatalf("accepted path %q did not normalize again: %v", normalized, err)
		}
		if again != normalized {
			t.Fatalf("normalization is not idempotent: first %q, second %q", normalized, again)
		}
	})
}
