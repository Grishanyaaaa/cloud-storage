package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// SoftDeleteNode marks a node and its subtree as deleted.
// Allowed for owner OR share-link with edit permission.
// Side effect: revokes any active share whose root is inside the deleted subtree.
func (s *StorageService) SoftDeleteNode(ctx context.Context, actor *port.Actor, nodeID string) error {
	id, err := valueobject.ParseNodeID(nodeID)
	if err != nil {
		return err
	}
	if actor == nil {
		return domainerr.ErrForbidden
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return err
	}
	if err := s.policy.allowDelete(actor, node); err != nil {
		return err
	}
	if node.IsRoot() {
		return domainerr.ErrRootImmutable
	}

	now := time.Now()
	if err := node.SoftDelete(now); err != nil {
		return err
	}

	return s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if _, err := s.nodeRepo.SoftDeleteSubtreeTx(ctx, tx, node, now); err != nil {
			return fmt.Errorf("soft delete subtree: %w", err)
		}
		if _, err := s.shareRepo.RevokeSubtreeTx(ctx, tx, node, now); err != nil {
			return fmt.Errorf("revoke shares of deleted subtree: %w", err)
		}
		return nil
	})
}
