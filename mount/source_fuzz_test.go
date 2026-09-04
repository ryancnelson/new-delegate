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
		"connect://example.com:443/",
		"connect://[::1]:443/",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_ = (Mount{Source: source, Target: "http://backend.internal/*"}).Validate()
	})
}

func FuzzConnectAuthorityValidationNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"example.com:443",
		"[::1]:443",
		"example.com",
		"example.com:65536",
		"user@example.com:443",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, authority string) {
		_ = ValidateConnectAuthority(authority)
	})
}
