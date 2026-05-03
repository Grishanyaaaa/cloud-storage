package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// GetNode returns a single node. Allowed for owner OR share-link with view permission.
func (s *StorageService) GetNode(ctx context.Context, actor *port.Actor, nodeID string) (*dto.NodeResponse, error) {
	if actor == nil {
		return nil, domainerr.ErrForbidden
	}
	id, err := valueobject.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	if err := s.policy.allowRead(actor, node); err != nil {
		return nil, err
	}
	if !node.IsFile() {
		return toNodeResponse(node, nil), nil
	}
	blob, err := s.blobRepo.GetByNodeID(ctx, node.ID())
	if err != nil {
		if errors.Is(err, domainerr.ErrFileBlobNotFound) {
			// File node without an attached blob (race during create) — return without size.
			return toNodeResponse(node, nil), nil
		}
		return nil, fmt.Errorf("get file blob: %w", err)
	}
	return toNodeResponse(node, blob), nil
}
