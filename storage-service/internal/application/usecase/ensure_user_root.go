package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

const defaultRootName = "root"

// EnsureUserRoot returns the user's root folder, creating it lazily on first call.
func (s *StorageService) EnsureUserRoot(ctx context.Context, actor *port.Actor) (*dto.NodeResponse, error) {
	if actor == nil || !actor.IsOwner() {
		return nil, domainerr.ErrForbidden
	}

	// Fast path: root already exists.
	if root, err := s.nodeRepo.GetRootByOwner(ctx, actor.UserID); err == nil {
		return toNodeResponse(root, nil), nil
	} else if !errors.Is(err, domainerr.ErrUserRootNotFound) && !errors.Is(err, domainerr.ErrNodeNotFound) {
		return nil, fmt.Errorf("get root by owner: %w", err)
	}

	// Slow path: create root + user_roots binding inside a single transaction.
	id := s.ids.NewNodeID()
	name, err := valueobject.NewNodeName(defaultRootName)
	if err != nil {
		return nil, err
	}
	root := entity.NewRootNode(id, actor.UserID, name, time.Now())
	binding := entity.NewUserRoot(actor.UserID, id, root.CreatedAt())

	txErr := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := s.nodeRepo.CreateTx(ctx, tx, root); err != nil {
			return err
		}
		if err := s.rootRepo.CreateTx(ctx, tx, binding); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		// In the unlikely race where two concurrent callers both insert,
		// the second will hit the unique-index violation; recover by reloading.
		if errors.Is(txErr, domainerr.ErrUserRootAlreadyExists) {
			if root, err := s.nodeRepo.GetRootByOwner(ctx, actor.UserID); err == nil {
				return toNodeResponse(root, nil), nil
			}
		}
		return nil, txErr
	}
	return toNodeResponse(root, nil), nil
}
