package entity

import "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"

// OperationResult is the per-operation outcome captured during ExecuteCommand.
// Stored alongside the command in `ai_commands.results`.
//
//   - Index — позиция операции в исходном плане (0-based).
//   - Success — true, если операция применилась к storage-service без ошибок.
//   - ErrorCode / ErrorMessage — заполняются только при Success=false. Это
//     transport-нейтральный код ошибки storage-service'а (например
//     "NODE_NOT_FOUND", "FORBIDDEN") — ai-service просто проксирует его.
type OperationResult struct {
	index        int
	kind         valueobject.OperationKind
	nodeID       valueobject.NodeID
	success      bool
	errorCode    string
	errorMessage string
}

// NewOperationResultSuccess builds a success result.
func NewOperationResultSuccess(
	index int,
	kind valueobject.OperationKind,
	nodeID valueobject.NodeID,
) OperationResult {
	return OperationResult{
		index:   index,
		kind:    kind,
		nodeID:  nodeID,
		success: true,
	}
}

// NewOperationResultFailure builds a failure result.
func NewOperationResultFailure(
	index int,
	kind valueobject.OperationKind,
	nodeID valueobject.NodeID,
	code string,
	message string,
) OperationResult {
	return OperationResult{
		index:        index,
		kind:         kind,
		nodeID:       nodeID,
		success:      false,
		errorCode:    code,
		errorMessage: message,
	}
}

// ReconstructOperationResult restores from persistence without revalidation.
func ReconstructOperationResult(
	index int,
	kind valueobject.OperationKind,
	nodeID valueobject.NodeID,
	success bool,
	errorCode string,
	errorMessage string,
) OperationResult {
	return OperationResult{
		index:        index,
		kind:         kind,
		nodeID:       nodeID,
		success:      success,
		errorCode:    errorCode,
		errorMessage: errorMessage,
	}
}

func (r OperationResult) Index() int                     { return r.index }
func (r OperationResult) Kind() valueobject.OperationKind { return r.kind }
func (r OperationResult) NodeID() valueobject.NodeID     { return r.nodeID }
func (r OperationResult) Success() bool                  { return r.success }
func (r OperationResult) ErrorCode() string              { return r.errorCode }
func (r OperationResult) ErrorMessage() string           { return r.errorMessage }
