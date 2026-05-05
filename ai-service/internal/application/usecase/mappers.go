package usecase

import (
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
)

// ToCommandResponse converts an AiCommand entity to its JSON-friendly response DTO.
func ToCommandResponse(c *entity.AiCommand) dto.CommandResponse {
	if c == nil {
		return dto.CommandResponse{}
	}
	return dto.CommandResponse{
		ID:           c.ID().String(),
		UserID:       c.UserID().String(),
		Input:        c.Input().String(),
		Plan:         toOperationDTOs(c.PlanOps()),
		Explanation:  c.Explanation(),
		Status:       c.Status().String(),
		LLMModel:     c.LLMModel(),
		LLMTokensIn:  c.LLMTokensIn(),
		LLMTokensOut: c.LLMTokensOut(),
		Results:      toOperationResultDTOs(c.Results()),
		CreatedAt:    c.CreatedAt(),
		ExpiresAt:    c.ExpiresAt(),
		ExecutedAt:   c.ExecutedAt(),
		CancelledAt:  c.CancelledAt(),
	}
}

func toOperationDTOs(ops []entity.Operation) []dto.OperationDTO {
	out := make([]dto.OperationDTO, 0, len(ops))
	for _, op := range ops {
		d := dto.OperationDTO{
			Kind:   op.Kind().String(),
			NodeID: op.NodeID().String(),
		}
		if op.Kind().IsRename() {
			d.NewName = op.NewName()
		}
		if op.Kind().IsMove() {
			if p := op.NewParentID(); p != nil {
				s := p.String()
				d.NewParentID = &s
			}
		}
		out = append(out, d)
	}
	return out
}

func toOperationResultDTOs(rs []entity.OperationResult) []dto.OperationResultDTO {
	if len(rs) == 0 {
		return nil
	}
	out := make([]dto.OperationResultDTO, 0, len(rs))
	for _, r := range rs {
		out = append(out, dto.OperationResultDTO{
			Index:        r.Index(),
			Kind:         r.Kind().String(),
			NodeID:       r.NodeID().String(),
			Success:      r.Success(),
			ErrorCode:    r.ErrorCode(),
			ErrorMessage: r.ErrorMessage(),
		})
	}
	return out
}
