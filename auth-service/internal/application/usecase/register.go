package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Register handles the registration of a new user.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 1. Валидация email через объект-значение
	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// 2. Валидация пароля через политику безопасности
	password, err := s.passwordPolicy.NewPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// 3. Проверка существования пользователя
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check user existence: %w", err)
	}
	if exists {
		return nil, domainerr.ErrUserAlreadyExists
	}

	// 4. Хеширование пароля
	hashedPassword, err := s.hasher.Hash(password.String())
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 5. Создание новой сущности пользователя
	now := time.Now()
	userID := valueobject.NewUserID()
	user := entity.NewUser(userID, email, hashedPassword, now)

	// 6. Сохранение в репозитории
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 7. Создание записи в логе аудита
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		userID,
		entity.ActionRegister,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Error("failed to save audit log", "error", err, "user_id", userID.String(), "action", entity.ActionRegister)
	}

	return &dto.RegisterResponse{
		UserID: userID.String(),
	}, nil
}
