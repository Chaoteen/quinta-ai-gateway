package common

import (
	"fmt"
	"os"
	"strings"
)

// ValidateProductionSecurity makes unsafe operating modes explicit startup errors.
// It does not provision TLS, network policies, backups, or secret storage.
func ValidateProductionSecurity() error {
	if !strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		return nil
	}
	for _, name := range []string{"SESSION_SECRET", "CRYPTO_SECRET", "BOOTSTRAP_TOKEN"} {
		if len(os.Getenv(name)) < 32 {
			return fmt.Errorf("production requires %s with at least 32 bytes", name)
		}
	}
	if os.Getenv("CRYPTO_SECRET") == os.Getenv("SESSION_SECRET") {
		return fmt.Errorf("production requires independent session and encryption secrets")
	}
	if !GetEnvOrDefaultBool("SESSION_COOKIE_SECURE", false) {
		return fmt.Errorf("production requires SESSION_COOKIE_SECURE=true and an HTTPS ingress")
	}
	for _, name := range []string{"DEBUG", "ENABLE_PPROF", "ENABLE_MOCK_PAYMENTS", "TLS_INSECURE_SKIP_VERIFY"} {
		if GetEnvOrDefaultBool(name, false) {
			return fmt.Errorf("%s must be disabled in production", name)
		}
	}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return fmt.Errorf("production outbound IP validation requires %s to be unset", name)
		}
	}
	return nil
}
