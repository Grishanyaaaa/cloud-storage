package security

import (
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
)

func NewPasswordPolicy(cfg config.SecurityConfig) valueobject.PasswordPolicy {
	return valueobject.PasswordPolicy{
		MinLength: cfg.MinPasswordLength,
		MaxLength: cfg.MaxPasswordLength,
	}
}
