package dto

import "strings"

// PlanCommandRequest is sent by the client to plan a command from natural language.
type PlanCommandRequest struct {
	Input string `json:"input"`
}

func (r *PlanCommandRequest) Validate() error {
	if strings.TrimSpace(r.Input) == "" {
		return ErrInputRequired
	}
	return nil
}

// ExecuteCommandRequest is sent to confirm and execute a previously-planned command.
// CommandID is taken from the URL path; the body is intentionally empty for now.
type ExecuteCommandRequest struct{}

// CancelCommandRequest is sent to mark a pending plan as cancelled.
// CommandID is taken from the URL path; the body is intentionally empty for now.
type CancelCommandRequest struct{}
