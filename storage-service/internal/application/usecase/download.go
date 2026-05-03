package usecase

import (
	"context"
	"fmt"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// GenerateDownloadURL returns a pre-signed GET URL for an active file blob.
// Allowed for owner OR share-link with view permission.
func (s *StorageService) GenerateDownloadURL(ctx context.Context, actor *port.Actor, req dto.GenerateDownloadURLRequest) (*dto.DownloadURLResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, domainerr.ErrForbidden
	}
	id, err := valueobject.ParseNodeID(req.NodeID)
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
		return nil, domainerr.ErrNodeKindMismatch
	}
	blob, err := s.blobRepo.GetByNodeID(ctx, node.ID())
	if err != nil {
		return nil, err
	}
	if !blob.IsActive() {
		return nil, domainerr.ErrFileNotActive
	}

	disposition := req.Disposition
	if disposition == "" {
		disposition = "attachment"
	}
	ttl := s.ttl.DownloadTTL(blob.Size())
	pre, err := s.storage.PresignDownload(ctx, port.PresignDownloadInput{
		Key:              blob.StorageKey(),
		TTL:              ttl,
		Filename:         node.Name().String(),
		Disposition:      disposition,
		ResponseMimeType: blob.MimeType().String(),
	})
	if err != nil {
		return nil, fmt.Errorf("presign download: %w", err)
	}
	return &dto.DownloadURLResponse{
		URL:       pre.URL,
		Method:    pre.Method,
		ExpiresAt: pre.ExpiresAt,
	}, nil
}
