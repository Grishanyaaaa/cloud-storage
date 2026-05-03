package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// RenameNode renames a node. Allowed for owner OR share-link with edit permission.
func (s *StorageService) RenameNode(ctx context.Context, actor *port.Actor, nodeID string, req dto.RenameNodeRequest) (*dto.NodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	id, err := valueobject.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	name, err := valueobject.NewNodeName(req.Name)
	if err != nil {
		return nil, err
	}

	// We always look up the node by its actual owner — share-links surface UserID = owner.
	if actor == nil {
		return nil, domainerr.ErrForbidden
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	if err := s.policy.allowRename(actor, node); err != nil {
		return nil, err
	}
	if err := node.Rename(name, time.Now()); err != nil {
		return nil, err
	}
	if err := s.nodeRepo.Rename(ctx, node); err != nil {
		if errors.Is(err, domainerr.ErrNodeNameTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("persist rename: %w", err)
	}
	return toNodeResponse(node, nil), nil
}
