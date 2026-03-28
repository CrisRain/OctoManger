package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const generatedKeyByteLength = 32

// Generate creates a URL-safe API key suitable for admin authentication.
func Generate() (string, error) {
	buf := make([]byte, generatedKeyByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "okm_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

// Hash returns a deterministic SHA-256 hex digest for the provided key.
func Hash(key string) string {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// Match checks whether the provided key matches the stored key hash.
func Match(providedKey, expectedHash string) bool {
	normalizedHash := strings.TrimSpace(expectedHash)
	if normalizedHash == "" {
		return false
	}
	providedHash := Hash(providedKey)
	if providedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(providedHash), []byte(normalizedHash)) == 1
}
