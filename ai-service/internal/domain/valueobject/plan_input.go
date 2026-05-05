package valueobject

import (
	"strings"
	"unicode/utf8"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
)

// PlanInput is the user-supplied natural-language command (Russian).
// We validate it as a UTF-8 string, with a max-length cap (in runes) and a
// non-empty (after trimming whitespace) constraint.
type PlanInput struct {
	value string
}

// NewPlanInput validates and constructs a PlanInput.
//   - maxChars must be > 0; ai-service config defaults this to 2000.
//   - trimmed input must be non-empty.
//   - input must be valid UTF-8.
func NewPlanInput(s string, maxChars int) (PlanInput, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return PlanInput{}, domainerr.ErrInputEmpty
	}
	if !utf8.ValidString(trimmed) {
		return PlanInput{}, domainerr.ErrInvalidPlanInput
	}
	if maxChars > 0 && utf8.RuneCountInString(trimmed) > maxChars {
		return PlanInput{}, domainerr.ErrInputTooLong
	}
	return PlanInput{value: trimmed}, nil
}

// PlanInputFromString wraps a raw string without validation
// (used only when reconstructing from persistence).
func PlanInputFromString(s string) PlanInput { return PlanInput{value: s} }

func (p PlanInput) String() string { return p.value }
func (p PlanInput) IsZero() bool   { return p.value == "" }
