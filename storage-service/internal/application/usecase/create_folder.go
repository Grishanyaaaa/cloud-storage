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
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// CreateFolder creates a new folder under ParentID. Owner-only.
func (s *StorageService) CreateFolder(ctx context.Context, actor *port.Actor, req dto.CreateFolderRequest) (*dto.NodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}

	parentID, err := valueobject.ParseNodeID(req.ParentID)
	if err != nil {
		return nil, err
	}
	name, err := valueobject.NewNodeName(req.Name)
	if err != nil {
		return nil, err
	}

	parent, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, parentID)
	if err != nil {
		return nil, err
	}
	if !parent.IsFolder() {
		return nil, domainerr.ErrNodeKindMismatch
	}
	if parent.IsDeleted() {
		return nil, domainerr.ErrNodeAlreadyDeleted
	}

	id := s.ids.NewNodeID()
	folder, err := entity.NewChildNode(id, actor.UserID, parent, valueobject.KindFolder, name, time.Now())
	if err != nil {
		return nil, err
	}

	if err := s.nodeRepo.Create(ctx, folder); err != nil {
		if errors.Is(err, domainerr.ErrNodeNameTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("create folder: %w", err)
	}
	return toNodeResponse(folder, nil), nil
}
