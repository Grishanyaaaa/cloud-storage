package dto

import "time"

// OperationDTO is the JSON-friendly view of a planned operation.
//
//	Kind: "delete" | "rename" | "move"
//	NewName: present only for rename.
//	NewParentID: present only for move.
type OperationDTO struct {
	Kind        string  `json:"kind"`
	NodeID      string  `json:"node_id"`
	NewName     string  `json:"new_name,omitempty"`
	NewParentID *string `json:"new_parent_id,omitempty"`
}

// OperationResultDTO is the per-operation outcome returned after Execute.
//
//	If Success=false, ErrorCode/ErrorMessage are set with the storage-service error.
type OperationResultDTO struct {
	Index        int    `json:"index"`
	Kind         string `json:"kind"`
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// CommandResponse is the canonical view of an ai_command returned over HTTP.
type CommandResponse struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	Input        string               `json:"input"`
	Plan         []OperationDTO       `json:"plan"`
	Explanation  string               `json:"explanation"`
	Status       string               `json:"status"`
	LLMModel     string               `json:"llm_model,omitempty"`
	LLMTokensIn  int                  `json:"llm_tokens_in,omitempty"`
	LLMTokensOut int                  `json:"llm_tokens_out,omitempty"`
	Results      []OperationResultDTO `json:"results,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	ExecutedAt   *time.Time           `json:"executed_at,omitempty"`
	CancelledAt  *time.Time           `json:"cancelled_at,omitempty"`
}
