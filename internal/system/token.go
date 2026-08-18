package system

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// TokenHasher hashes and verifies opaque access tokens. Tokens are never
// persisted in plain text.
type TokenHasher interface {
	Hash(token string) string
	Generate() (string, string, error)
}

// SHA256TokenHasher is the default hasher used by the service.
type SHA256TokenHasher struct{}

func (SHA256TokenHasher) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (SHA256TokenHasher) Generate() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	token := "tkn_" + base64.RawURLEncoding.EncodeToString(raw)
	return token, SHA256TokenHasher{}.Hash(token), nil
}
