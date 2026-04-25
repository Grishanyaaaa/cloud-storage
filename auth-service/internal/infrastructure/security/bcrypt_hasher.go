package security

import (
	"errors"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cfg config.SecurityConfig) *BcryptHasher {
	return &BcryptHasher{cost: cfg.BcryptCost}
}

// Hash — bcrypt сам генерит соль, повторный вызов даст другой хеш и это нормально
func (h *BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare возвращаем bool а не error наружу, чтобы сервисный слой не парсил bcrypt-ошибки
func (h *BcryptHasher) Compare(hashedPassword, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
