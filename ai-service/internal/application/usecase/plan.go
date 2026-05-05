package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Plan accepts a natural-language input, asks the LLM for a JSON operation
// plan, validates it against the user's tree, and persists the result as a
// new ai_command in `awaiting_confirmation` state.
//
// On success the returned AiCommand has:
//   - status = awaiting_confirmation
//   - planOps populated (possibly empty if LLM declined)
//   - explanation populated
//   - expiresAt = createdAt + planTTL
func (s *AIService) Plan(ctx context.Context, actor *port.Actor, input string) (*entity.AiCommand, error) {
	if actor == nil || !actor.IsOwner() {
		return nil, domainerr.ErrForbidden
	}

	// 1. Validate input.
	planInput, err := valueobject.NewPlanInput(input, s.maxInputChars)
	if err != nil {
		return nil, err
	}

	// 2. Fetch user tree from storage-service.
	tree, err := s.storage.GetTree(ctx, actor.JWT, 0, 0)
	if err != nil {
		return nil, err
	}

	// 3. Find user's root in the tree (depth=1, parent=nil).
	rootID := findRootID(tree)

	// 4. Call LLM with retry on parse/validation failure.
	systemMsg, userMsg, schema := s.prompt.Build(tree, rootID, planInput.String(), 0)
	llmReq := port.LLMRequest{
		SystemPrompt: systemMsg,
		UserMessage:  userMsg,
		JSONSchema:   schema,
	}

	var (
		ops         []entity.Operation
		explanation string
		llmResp     port.LLMResponse
		lastErr     error
	)

	maxAttempts := s.maxRetries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		llmResp, err = s.llm.Complete(ctx, llmReq)
		if err != nil {
			s.logger.WarnContext(ctx, "llm complete failed",
				slogString("user_id", actor.UserID.String()),
				slogInt("attempt", attempt),
				slogError(err))
			return nil, err
		}
		ops, explanation, err = s.parser.Parse(llmResp.Text)
		if err != nil {
			lastErr = err
			s.logger.WarnContext(ctx, "llm response parse failed",
				slogString("user_id", actor.UserID.String()),
				slogInt("attempt", attempt),
				slogError(err))
			continue
		}
		if err := s.validator.Validate(ops, tree); err != nil {
			lastErr = err
			s.logger.WarnContext(ctx, "llm plan validation failed",
				slogString("user_id", actor.UserID.String()),
				slogInt("attempt", attempt),
				slogError(err))
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		// Convert any parser/validator failure into a single domain error
		// (LLM_INVALID_RESPONSE / INVALID_PLAN already wraps the cause).
		var de *domainerr.DomainError
		if errors.As(lastErr, &de) {
			return nil, de
		}
		return nil, domainerr.New(
			domainerr.CodeLLMInvalidResponse,
			fmt.Sprintf("LLM produced an unusable response after %d attempt(s)", maxAttempts),
			lastErr,
		)
	}

	// 5. Persist as awaiting_confirmation.
	now := s.now()
	expiresAt := now.Add(s.planTTL)

	cmd := entity.NewAiCommand(
		s.ids.NewCommandID(),
		actor.UserID,
		planInput,
		ops,
		explanation,
		llmResp.Model,
		llmResp.TokensIn,
		llmResp.TokensOut,
		now,
		expiresAt,
	)

	if err := s.cmdRepo.Create(ctx, cmd); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "ai command planned",
		slogString("user_id", actor.UserID.String()),
		slogString("command_id", cmd.ID().String()),
		slogInt("ops", len(ops)),
		slogInt("tokens_in", llmResp.TokensIn),
		slogInt("tokens_out", llmResp.TokensOut))
	return cmd, nil
}

func findRootID(tree []port.TreeNode) valueobject.NodeID {
	for _, n := range tree {
		if n.ParentID == nil || n.Depth == 1 {
			return n.ID
		}
	}
	return valueobject.NodeID{}
}
