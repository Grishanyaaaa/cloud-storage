package usecase

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Get returns a single ai_command, asserting ownership.
//
// Note: this method does NOT mutate state. If the command's TTL has elapsed
// but the janitor hasn't expired it yet, callers will see status=awaiting_confirmation.
// Execute/Cancel will reject it with ErrCommandExpired in that case.
func (s *AIService) Get(ctx context.Context, actor *port.Actor, id valueobject.CommandID) (*entity.AiCommand, error) {
	if actor == nil || !actor.IsOwner() {
		return nil, domainerr.ErrForbidden
	}

	cmd, err := s.cmdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}
