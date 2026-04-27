package usecase

import (
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
)

// AuthService implements the AuthUseCase interface.
// It coordinates domain entities and infrastructure services.
type AuthService struct {
	userRepo     repository.UserRepository
	tokenRepo    repository.RefreshTokenRepository
	auditRepo    repository.AuditLogRepository
	hasher       port.PasswordHasher
	tokenManager port.TokenManager
	tokenHasher  port.TokenHasher
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	auditRepo repository.AuditLogRepository,
	hasher port.PasswordHasher,
	tokenManager port.TokenManager,
	tokenHasher port.TokenHasher,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		auditRepo:    auditRepo,
		hasher:       hasher,
		tokenManager: tokenManager,
		tokenHasher:  tokenHasher,
	}
}
