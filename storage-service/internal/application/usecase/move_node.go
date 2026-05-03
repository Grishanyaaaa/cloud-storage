package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// MoveNode reparents a node. Owner-only — moving across the share root would
// inherently require permissions beyond what a share-link grants.
func (s *StorageService) MoveNode(ctx context.Context, actor *port.Actor, nodeID string, req dto.MoveNodeRequest) (*dto.NodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}
	id, err := valueobject.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	newParentID, err := valueobject.ParseNodeID(req.NewParentID)
	if err != nil {
		return nil, err
	}

	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	if node.IsRoot() {
		return nil, domainerr.ErrRootImmutable
	}
	newParent, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, newParentID)
	if err != nil {
		return nil, err
	}

	oldPath := node.Path()
	if err := node.MoveTo(newParent, time.Now()); err != nil {
		return nil, err
	}

	txErr := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := s.nodeRepo.MoveTx(ctx, tx, oldPath, node); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, domainerr.ErrNodeNameTaken) {
			return nil, txErr
		}
		return nil, fmt.Errorf("persist move: %w", txErr)
	}
	return toNodeResponse(node, nil), nil
}
