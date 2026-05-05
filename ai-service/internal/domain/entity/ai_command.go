package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// AiCommand is the aggregate root of the ai-service.
//
// Lifecycle:
//
//	NewAiCommand → status=awaiting_confirmation
//	  CanExecute(now)  → returns ErrCommandExpired / ErrCommandAlreadyCancelled / ...
//	  MarkExecuted([]OperationResult)  → executed (or failed if any op failed)
//	  MarkCancelled                    → cancelled (only from awaiting_confirmation)
//	  MarkExpired                      → expired   (only from awaiting_confirmation, by janitor)
type AiCommand struct {
	id           valueobject.CommandID
	userID       valueobject.UserID
	input        valueobject.PlanInput
	planOps      []Operation
	explanation  string
	status       valueobject.CommandStatus
	llmModel     string
	llmTokensIn  int
	llmTokensOut int
	results      []OperationResult
	createdAt    time.Time
	expiresAt    time.Time
	executedAt   *time.Time
	cancelledAt  *time.Time
}

// NewAiCommand creates a fresh ai_command in awaiting_confirmation state.
//
// Inputs are trusted (caller — use case — already validated PlanInput).
// `planOps` may be empty (e.g. LLM declined / asked for clarification);
// in that case ExecuteCommand will simply produce an empty results array.
func NewAiCommand(
	id valueobject.CommandID,
	userID valueobject.UserID,
	input valueobject.PlanInput,
	planOps []Operation,
	explanation string,
	llmModel string,
	llmTokensIn int,
	llmTokensOut int,
	createdAt time.Time,
	expiresAt time.Time,
) *AiCommand {
	return &AiCommand{
		id:           id,
		userID:       userID,
		input:        input,
		planOps:      planOps,
		explanation:  explanation,
		status:       valueobject.CommandStatusAwaitingConfirmation,
		llmModel:     llmModel,
		llmTokensIn:  llmTokensIn,
		llmTokensOut: llmTokensOut,
		results:      nil,
		createdAt:    createdAt,
		expiresAt:    expiresAt,
		executedAt:   nil,
		cancelledAt:  nil,
	}
}

// ReconstructAiCommand rebuilds an AiCommand from persisted state (no validation).
func ReconstructAiCommand(
	id valueobject.CommandID,
	userID valueobject.UserID,
	input valueobject.PlanInput,
	planOps []Operation,
	explanation string,
	status valueobject.CommandStatus,
	llmModel string,
	llmTokensIn int,
	llmTokensOut int,
	results []OperationResult,
	createdAt time.Time,
	expiresAt time.Time,
	executedAt *time.Time,
	cancelledAt *time.Time,
) *AiCommand {
	return &AiCommand{
		id:           id,
		userID:       userID,
		input:        input,
		planOps:      planOps,
		explanation:  explanation,
		status:       status,
		llmModel:     llmModel,
		llmTokensIn:  llmTokensIn,
		llmTokensOut: llmTokensOut,
		results:      results,
		createdAt:    createdAt,
		expiresAt:    expiresAt,
		executedAt:   executedAt,
		cancelledAt:  cancelledAt,
	}
}

// Getters.
func (c *AiCommand) ID() valueobject.CommandID         { return c.id }
func (c *AiCommand) UserID() valueobject.UserID        { return c.userID }
func (c *AiCommand) Input() valueobject.PlanInput      { return c.input }
func (c *AiCommand) PlanOps() []Operation              { return c.planOps }
func (c *AiCommand) Explanation() string               { return c.explanation }
func (c *AiCommand) Status() valueobject.CommandStatus { return c.status }
func (c *AiCommand) LLMModel() string                  { return c.llmModel }
func (c *AiCommand) LLMTokensIn() int                  { return c.llmTokensIn }
func (c *AiCommand) LLMTokensOut() int                 { return c.llmTokensOut }
func (c *AiCommand) Results() []OperationResult        { return c.results }
func (c *AiCommand) CreatedAt() time.Time              { return c.createdAt }
func (c *AiCommand) ExpiresAt() time.Time              { return c.expiresAt }
func (c *AiCommand) ExecutedAt() *time.Time            { return c.executedAt }
func (c *AiCommand) CancelledAt() *time.Time           { return c.cancelledAt }

// IsExpired returns true if `now` is past expiresAt and the plan was never executed/cancelled.
func (c *AiCommand) IsExpired(now time.Time) bool {
	return c.status.IsAwaitingConfirmation() && !now.Before(c.expiresAt)
}

// AssertOwner returns ErrCommandForbidden if the command does not belong to user.
func (c *AiCommand) AssertOwner(user valueobject.UserID) error {
	if !c.userID.Equals(user) {
		return domainerr.ErrCommandForbidden
	}
	return nil
}

// CanExecute checks whether the command is in a state that allows execution.
//
// Possible failure modes:
//   - already executed / failed   → ErrCommandAlreadyExecuted
//   - already cancelled           → ErrCommandAlreadyCancelled
//   - already expired (by status) → ErrCommandExpired
//   - awaiting but past TTL       → ErrCommandExpired
func (c *AiCommand) CanExecute(now time.Time) error {
	if c.status.IsExecuted() || c.status.IsFailed() {
		return domainerr.ErrCommandAlreadyExecuted
	}
	if c.status.IsCancelled() {
		return domainerr.ErrCommandAlreadyCancelled
	}
	if c.status.IsExpired() {
		return domainerr.ErrCommandExpired
	}
	if c.IsExpired(now) {
		return domainerr.ErrCommandExpired
	}
	return nil
}

// CanCancel checks whether the command can transition to cancelled.
func (c *AiCommand) CanCancel(now time.Time) error {
	if c.status.IsExecuted() || c.status.IsFailed() {
		return domainerr.ErrCommandAlreadyExecuted
	}
	if c.status.IsCancelled() {
		return domainerr.ErrCommandAlreadyCancelled
	}
	if c.status.IsExpired() || c.IsExpired(now) {
		return domainerr.ErrCommandExpired
	}
	return nil
}

// MarkExecuted transitions to executed (or failed if at least one op failed).
// Stores per-op results and bumps executedAt.
func (c *AiCommand) MarkExecuted(results []OperationResult, now time.Time) {
	c.results = results
	c.executedAt = &now
	if anyFailed(results) {
		c.status = valueobject.CommandStatusFailed
	} else {
		c.status = valueobject.CommandStatusExecuted
	}
}

// MarkCancelled transitions to cancelled.
// Caller is responsible for invoking CanCancel(now) first.
func (c *AiCommand) MarkCancelled(now time.Time) {
	c.status = valueobject.CommandStatusCancelled
	c.cancelledAt = &now
}

// MarkExpired transitions to expired (intended for janitor calls).
// Idempotent — no-op if status is no longer awaiting_confirmation.
func (c *AiCommand) MarkExpired() {
	if c.status.IsAwaitingConfirmation() {
		c.status = valueobject.CommandStatusExpired
	}
}

func anyFailed(rs []OperationResult) bool {
	for _, r := range rs {
		if !r.Success() {
			return true
		}
	}
	return false
}
