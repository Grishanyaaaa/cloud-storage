package usecase

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// RevokeShareLink marks a share as revoked. Owner-only.
func (s *StorageService) RevokeShareLink(ctx context.Context, actor *port.Actor, req dto.RevokeShareRequest) error {
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return err
	}
	id, err := valueobject.ParseShareID(req.ShareID)
	if err != nil {
		return err
	}
	return s.shareRepo.RevokeByID(ctx, actor.UserID, id, time.Now())
}
