package common

import (
	"strings"
	"testing"
)

func TestProductionSecurityRejectsUnsafeModes(t *testing.T) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", strings.Repeat("s", 32))
	t.Setenv("CRYPTO_SECRET", strings.Repeat("c", 32))
	t.Setenv("BOOTSTRAP_TOKEN", strings.Repeat("b", 32))
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	for _, key := range []string{"DEBUG", "ENABLE_PPROF", "ENABLE_MOCK_PAYMENTS", "TLS_INSECURE_SKIP_VERIFY"} {
		t.Setenv(key, "false")
	}
	if err := ValidateProductionSecurity(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"DEBUG", "ENABLE_PPROF", "ENABLE_MOCK_PAYMENTS", "TLS_INSECURE_SKIP_VERIFY"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "true")
			if err := ValidateProductionSecurity(); err == nil {
				t.Fatal("unsafe production mode accepted")
			}
		})
	}
	t.Setenv("SESSION_SECRET", "short")
	if err := ValidateProductionSecurity(); err == nil {
		t.Fatal("weak session secret accepted")
	}
}

func TestSecureRandomStringRetainsAlphabetAndLength(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	value := GetRandomString(1024)
	if len(value) != 1024 {
		t.Fatal("wrong random string length")
	}
	for _, c := range value {
		if !strings.ContainsRune(alphabet, c) {
			t.Fatal("unexpected random string alphabet")
		}
	}
}
