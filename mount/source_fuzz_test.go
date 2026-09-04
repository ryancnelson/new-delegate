package mount

import "testing"

func FuzzURLSourceValidationNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"http://example.com/*",
		"https://example.com:8443/docs/*",
		"http://user@example.com/*",
		"http://[::1]:8080/*",
		"not a URL",
		"http://example.com/a%2fb/*",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_ = (Mount{Source: source, Target: "http://backend.internal/*"}).Validate()
	})
}
