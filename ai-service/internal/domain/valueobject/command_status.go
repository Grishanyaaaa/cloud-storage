package valueobject

import "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"

// CommandStatus enumerates the lifecycle of an ai_command record.
//
//	awaiting_confirmation → executed
//	awaiting_confirmation → cancelled
//	awaiting_confirmation → expired (by janitor)
//	executed              → failed (if at least one op failed during execution)
//
// `failed` is a *terminal* status — the plan was attempted but at least one
// operation could not be applied. For MVP we use stop-at-first-failure.
type CommandStatus struct {
	value string
}

const (
	statusAwaitingConfirmation = "awaiting_confirmation"
	statusExecuted             = "executed"
	statusFailed               = "failed"
	statusCancelled            = "cancelled"
	statusExpired              = "expired"
)

var (
	CommandStatusAwaitingConfirmation = CommandStatus{value: statusAwaitingConfirmation}
	CommandStatusExecuted             = CommandStatus{value: statusExecuted}
	CommandStatusFailed               = CommandStatus{value: statusFailed}
	CommandStatusCancelled            = CommandStatus{value: statusCancelled}
	CommandStatusExpired              = CommandStatus{value: statusExpired}
)

// ParseCommandStatus validates a string against the allowed enum.
func ParseCommandStatus(s string) (CommandStatus, error) {
	switch s {
	case statusAwaitingConfirmation:
		return CommandStatusAwaitingConfirmation, nil
	case statusExecuted:
		return CommandStatusExecuted, nil
	case statusFailed:
		return CommandStatusFailed, nil
	case statusCancelled:
		return CommandStatusCancelled, nil
	case statusExpired:
		return CommandStatusExpired, nil
	default:
		return CommandStatus{}, domainerr.ErrInvalidCommandStatus
	}
}

func (s CommandStatus) String() string { return s.value }
func (s CommandStatus) IsZero() bool   { return s.value == "" }

func (s CommandStatus) IsAwaitingConfirmation() bool {
	return s.value == statusAwaitingConfirmation
}
func (s CommandStatus) IsExecuted() bool  { return s.value == statusExecuted }
func (s CommandStatus) IsFailed() bool    { return s.value == statusFailed }
func (s CommandStatus) IsCancelled() bool { return s.value == statusCancelled }
func (s CommandStatus) IsExpired() bool   { return s.value == statusExpired }

// IsTerminal returns true if the status cannot transition further.
func (s CommandStatus) IsTerminal() bool {
	switch s.value {
	case statusExecuted, statusFailed, statusCancelled, statusExpired:
		return true
	}
	return false
}

func (s CommandStatus) Equals(other CommandStatus) bool { return s.value == other.value }
