package usecase

import (
	"context"
	"errors"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Execute applies the previously-planned operations of `id` to storage-service.
//
// Semantics (MVP):
//   - The command must be in awaiting_confirmation status and not expired.
//   - Operations are applied in array order. On the first failure, remaining
//     operations are NOT attempted (stop-at-first-failure). All preceding
//     successes are still recorded in `results`.
//   - The aggregate status becomes `executed` if all ops succeeded, otherwise `failed`.
//   - The whole transition (load → run ops → persist) runs inside a single DB
//     transaction with FOR UPDATE on the row to make execute idempotent under
//     concurrent requests.
func (s *AIService) Execute(ctx context.Context, actor *port.Actor, id valueobject.CommandID) (*entity.AiCommand, error) {
	if actor == nil || !actor.IsOwner() {
		return nil, domainerr.ErrForbidden
	}

	var executed *entity.AiCommand
	err := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		cmd, err := s.cmdRepo.GetByIDTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := s.policy.allowOwner(actor, cmd); err != nil {
			return err
		}
		now := s.now()
		if err := cmd.CanExecute(now); err != nil {
			return err
		}

		results := s.applyOps(ctx, actor.JWT, cmd.PlanOps())
		cmd.MarkExecuted(results, s.now())

		if err := s.cmdRepo.UpdateTx(ctx, tx, cmd); err != nil {
			return err
		}
		executed = cmd
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "ai command executed",
		slogString("user_id", actor.UserID.String()),
		slogString("command_id", executed.ID().String()),
		slogString("status", executed.Status().String()),
		slogInt("ops", len(executed.PlanOps())),
		slogInt("results", len(executed.Results())))
	return executed, nil
}

// applyOps executes operations sequentially, stopping at the first failure.
// All preceding successes are still returned in the results slice; the failed
// op is the last entry. Subsequent ops are not attempted.
func (s *AIService) applyOps(ctx context.Context, jwt string, ops []entity.Operation) []entity.OperationResult {
	out := make([]entity.OperationResult, 0, len(ops))
	for i, op := range ops {
		var err error
		switch {
		case op.Kind().IsDelete():
			err = s.storage.DeleteNode(ctx, jwt, op.NodeID())
		case op.Kind().IsRename():
			err = s.storage.RenameNode(ctx, jwt, op.NodeID(), op.NewName())
		case op.Kind().IsMove():
			parent := op.NewParentID()
			if parent == nil {
				err = domainerr.ErrInvalidOperation
			} else {
				err = s.storage.MoveNode(ctx, jwt, op.NodeID(), *parent)
			}
		default:
			err = domainerr.ErrInvalidOperationKind
		}

		if err != nil {
			code, msg := extractErrorCode(err)
			out = append(out, entity.NewOperationResultFailure(i, op.Kind(), op.NodeID(), code, msg))
			return out
		}
		out = append(out, entity.NewOperationResultSuccess(i, op.Kind(), op.NodeID()))
	}
	return out
}

// extractErrorCode pulls the storage-service domain code out of an error if it
// is wrapped as DomainError; otherwise returns a generic UPSTREAM_ERROR code.
func extractErrorCode(err error) (string, string) {
	var de *domainerr.DomainError
	if errors.As(err, &de) {
		return de.Code, de.Message
	}
	return "UPSTREAM_ERROR", err.Error()
}
