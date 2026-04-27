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

	// 2. Проверка существования пользователя
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check user existence: %w", err)
	}
	if exists {
		return nil, domainerr.ErrUserAlreadyExists
	}

	// 3. Хеширование пароля
	// Мы передаем сырой пароль в hasher, который инкапсулирует алгоритм (например, bcrypt)
	hashedPassword, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 4. Создание новой сущности пользователя
	now := time.Now()
	userID := valueobject.NewUserID()
	user := entity.NewUser(userID, email, hashedPassword, now)

	// 5. Сохранение в репозитории
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 6. Создание записи в логе аудита
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		userID,
		entity.ActionRegister,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		// Мы не прерываем регистрацию, если лог аудита не сохранился,
		// но стоит хотя бы залогировать ошибку.
		// log.Error("failed to create audit log", err)
	}

	return &dto.RegisterResponse{
		UserID: userID.String(),
	}, nil
}
