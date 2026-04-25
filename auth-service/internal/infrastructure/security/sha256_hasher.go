package security

import (
	"crypto/sha256"
	"encoding/hex"
)

type SHA256TokenHasher struct{}

func NewSHA256TokenHasher() *SHA256TokenHasher {
	return &SHA256TokenHasher{}
}

// Hash SHA-256 детерминированный, один и тот же токен всегда даст один хеш
// это нужно чтобы при валидации сравнить хеш из базы с хешем входящего токена
func (h *SHA256TokenHasher) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
