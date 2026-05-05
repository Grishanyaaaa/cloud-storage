package usecase

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Cancel transitions an awaiting_confirmation command into cancelled.
// Idempotency: if the command is already cancelled, returns ErrCommandAlreadyCancelled.
func (s *AIService) Cancel(ctx context.Context, actor *port.Actor, id valueobject.CommandID) (*entity.AiCommand, error) {
	if actor == nil || !actor.IsOwner() {
		return nil, domainerr.ErrForbidden
	}

	var cancelled *entity.AiCommand
	err := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		cmd, err := s.cmdRepo.GetByIDTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := s.policy.allowOwner(actor, cmd); err != nil {
			return err
		}
		now := s.now()
		if err := cmd.CanCancel(now); err != nil {
			return err
		}

		cmd.MarkCancelled(now)
		if err := s.cmdRepo.UpdateTx(ctx, tx, cmd); err != nil {
			return err
		}
		cancelled = cmd
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "ai command cancelled",
		slogString("user_id", actor.UserID.String()),
		slogString("command_id", cancelled.ID().String()))
	return cancelled, nil
}
