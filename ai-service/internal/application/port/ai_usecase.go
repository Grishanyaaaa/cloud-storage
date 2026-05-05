package port

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// AIUseCase is the application-level entry point used by HTTP handlers.
// It wraps the four operations of the ai-service:
//
//	Plan    — accept NL input, ask LLM, validate, persist as awaiting_confirmation.
//	Execute — apply a previously-planned command via storage-service (stop-at-first-failure).
//	Cancel  — mark a pending plan as cancelled.
//	Get     — fetch a single command (status / plan / results).
type AIUseCase interface {
	Plan(ctx context.Context, actor *Actor, input string) (*entity.AiCommand, error)
	Execute(ctx context.Context, actor *Actor, id valueobject.CommandID) (*entity.AiCommand, error)
	Cancel(ctx context.Context, actor *Actor, id valueobject.CommandID) (*entity.AiCommand, error)
	Get(ctx context.Context, actor *Actor, id valueobject.CommandID) (*entity.AiCommand, error)
}
