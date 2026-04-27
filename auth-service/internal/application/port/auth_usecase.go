package port

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
)

// AuthUseCase defines the input port for authentication and registration operations.
type AuthUseCase interface {
	// Register handles new user registration.
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)

	// Login handles user authentication and returns a token pair.
	Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPairResponse, error)

	// Refresh handles token refresh using a refresh token.
	Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.TokenPairResponse, error)

	// Logout handles user logout by revoking the refresh token.
	Logout(ctx context.Context, req dto.LogoutRequest) error
}
