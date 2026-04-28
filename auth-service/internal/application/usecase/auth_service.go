package usecase

import (
	"log/slog"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// AuthService implements the AuthUseCase interface.
// It coordinates domain entities and infrastructure services.
type AuthService struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.RefreshTokenRepository
	auditRepo      repository.AuditLogRepository
	hasher         port.PasswordHasher
	tokenManager   port.TokenManager
	tokenHasher    port.TokenHasher
	passwordPolicy valueobject.PasswordPolicy
	logger         *slog.Logger
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	auditRepo repository.AuditLogRepository,
	hasher port.PasswordHasher,
	tokenManager port.TokenManager,
	tokenHasher port.TokenHasher,
	passwordPolicy valueobject.PasswordPolicy,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		auditRepo:      auditRepo,
		hasher:         hasher,
		tokenManager:   tokenManager,
		tokenHasher:    tokenHasher,
		passwordPolicy: passwordPolicy,
		logger:         logger,
	}
}
