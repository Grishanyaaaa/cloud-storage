package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// rawLLMResponse mirrors the JSON Schema returned by the LLM.
type rawLLMResponse struct {
	Ops         []rawLLMOperation `json:"ops"`
	Explanation string            `json:"explanation"`
}

type rawLLMOperation struct {
	Kind        string  `json:"kind"`
	NodeID      string  `json:"node_id"`
	NewName     string  `json:"new_name,omitempty"`
	NewParentID *string `json:"new_parent_id,omitempty"`
}

// llmResponseParser converts the raw LLM text into domain Operation entities.
type llmResponseParser struct{}

func newLLMResponseParser() *llmResponseParser { return &llmResponseParser{} }

// Parse extracts the JSON object from the LLM text and converts it.
//
// The LLM is instructed to return ONLY valid JSON, but providers occasionally
// wrap it in ```json … ``` fences or add stray characters around it. We
// tolerate both:
//   - strip a leading/trailing markdown code fence;
//   - if that still fails, slice from the first '{' to the last '}'.
//
// Returns (ops, explanation) or domainerr.ErrLLMInvalidResponse on any failure.
func (p *llmResponseParser) Parse(text string) ([]entity.Operation, string, error) {
	cleaned := p.cleanText(text)
	if cleaned == "" {
		return nil, "", domainerr.ErrLLMInvalidResponse
	}

	var raw rawLLMResponse
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, "", domainerr.New(
			domainerr.CodeLLMInvalidResponse,
			fmt.Sprintf("LLM returned invalid JSON: %s", err.Error()),
			err,
		)
	}

	ops := make([]entity.Operation, 0, len(raw.Ops))
	for i, ro := range raw.Ops {
		op, err := p.parseOperation(ro)
		if err != nil {
			return nil, "", domainerr.New(
				domainerr.CodeLLMInvalidResponse,
				fmt.Sprintf("LLM operation #%d invalid: %s", i, err.Error()),
				err,
			)
		}
		ops = append(ops, op)
	}
	return ops, raw.Explanation, nil
}

func (p *llmResponseParser) cleanText(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return ""
	}
	// Strip ```json … ``` fences if present.
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```json")
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSuffix(t, "```")
		t = strings.TrimSpace(t)
	}
	// As a last resort, slice from the first '{' to the last '}'.
	if !strings.HasPrefix(t, "{") || !strings.HasSuffix(t, "}") {
		first := strings.Index(t, "{")
		last := strings.LastIndex(t, "}")
		if first >= 0 && last > first {
			t = t[first : last+1]
		}
	}
	return t
}

func (p *llmResponseParser) parseOperation(ro rawLLMOperation) (entity.Operation, error) {
	kind, err := valueobject.ParseOperationKind(ro.Kind)
	if err != nil {
		return entity.Operation{}, err
	}
	nodeID, err := valueobject.ParseNodeID(ro.NodeID)
	if err != nil {
		return entity.Operation{}, err
	}
	switch {
	case kind.IsDelete():
		return entity.NewDeleteOperation(nodeID)
	case kind.IsRename():
		newName := strings.TrimSpace(ro.NewName)
		if newName == "" {
			return entity.Operation{}, domainerr.ErrInvalidOperation
		}
		return entity.NewRenameOperation(nodeID, newName)
	case kind.IsMove():
		if ro.NewParentID == nil || *ro.NewParentID == "" {
			return entity.Operation{}, domainerr.ErrInvalidOperation
		}
		newParent, err := valueobject.ParseNodeID(*ro.NewParentID)
		if err != nil {
			return entity.Operation{}, err
		}
		return entity.NewMoveOperation(nodeID, newParent)
	default:
		return entity.Operation{}, domainerr.ErrInvalidOperationKind
	}
}
