package accountapp

import (
	"strings"

	accountdomain "octomanger/internal/domains/accounts/domain"
)

func verificationStatusForAction(action string, success bool) (string, bool) {
	if !isVerificationAction(action) {
		return "", false
	}
	if success {
		return accountdomain.StatusActive, true
	}
	return accountdomain.StatusInactive, true
}

func isVerificationAction(action string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(action))
	if normalized == "" {
		return false
	}
	return normalized == "VERIFY" ||
		strings.Contains(normalized, "VERIFY") ||
		strings.Contains(normalized, "VALIDATE")
}
