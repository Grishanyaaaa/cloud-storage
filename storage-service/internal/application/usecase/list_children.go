package usecase

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ListChildren returns alive children of a folder. Allowed for owner OR share-link.
// IncludeDeleted is honored only when actor is the owner.
func (s *StorageService) ListChildren(ctx context.Context, actor *port.Actor, req dto.ListChildrenRequest) (*dto.ListChildrenResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, domainerr.ErrForbidden
	}
	parentID, err := valueobject.ParseNodeID(req.ParentID)
	if err != nil {
		return nil, err
	}
	parent, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, parentID)
	if err != nil {
		return nil, err
	}
	if err := s.policy.allowRead(actor, parent); err != nil {
		return nil, err
	}
	if !parent.IsFolder() {
		return nil, domainerr.ErrNodeKindMismatch
	}
	includeDeleted := req.IncludeDeleted && actor.IsOwner()
	items, next, err := s.nodeRepo.ListChildren(ctx, actor.UserID, parentID, repository.NodeFilter{
		IncludeDeleted: includeDeleted,
		Cursor:         req.Cursor,
		Limit:          req.Limit,
	})
	if err != nil {
		return nil, err
	}
	resp := &dto.ListChildrenResponse{
		Items:      make([]dto.NodeResponse, 0, len(items)),
		NextCursor: next,
	}
	for _, n := range items {
		resp.Items = append(resp.Items, *toNodeResponse(n, nil))
	}
	return resp, nil
}
