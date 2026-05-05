package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Compile-time check: PostgresAiCommandRepository implements repository.AiCommandRepository
var _ repository.AiCommandRepository = (*PostgresAiCommandRepository)(nil)

// PostgresAiCommandRepository persists AiCommand aggregates in PostgreSQL.
type PostgresAiCommandRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAiCommandRepository creates a new repository instance.
func NewPostgresAiCommandRepository(pool *pgxpool.Pool) *PostgresAiCommandRepository {
	return &PostgresAiCommandRepository{pool: pool}
}

// jsonOperation is the on-the-wire (JSONB) representation of entity.Operation.
type jsonOperation struct {
	Kind        string  `json:"kind"`
	NodeID      string  `json:"node_id"`
	NewName     string  `json:"new_name,omitempty"`
	NewParentID *string `json:"new_parent_id,omitempty"`
}

// jsonOperationResult is the on-the-wire (JSONB) representation of entity.OperationResult.
type jsonOperationResult struct {
	Index        int    `json:"index"`
	Kind         string `json:"kind"`
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Create inserts a new ai_command row.
func (r *PostgresAiCommandRepository) Create(ctx context.Context, cmd *entity.AiCommand) error {
	planJSON, err := encodePlan(cmd.PlanOps())
	if err != nil {
		return err
	}
	resultsJSON, err := encodeResults(cmd.Results())
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO ai_commands (
			id, user_id, input, plan_ops, explanation, status,
			llm_model, llm_tokens_in, llm_tokens_out, results,
			created_at, expires_at, executed_at, cancelled_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14
		)`
	_, err = r.pool.Exec(ctx, q,
		cmd.ID().Value(),
		cmd.UserID().Value(),
		cmd.Input().String(),
		planJSON,
		cmd.Explanation(),
		cmd.Status().String(),
		cmd.LLMModel(),
		cmd.LLMTokensIn(),
		cmd.LLMTokensOut(),
		resultsJSON,
		cmd.CreatedAt(),
		cmd.ExpiresAt(),
		cmd.ExecutedAt(),
		cmd.CancelledAt(),
	)
	if err != nil {
		return fmt.Errorf("insert ai_command: %w", err)
	}
	return nil
}

// GetByID retrieves a single command by ID using the connection pool.
func (r *PostgresAiCommandRepository) GetByID(
	ctx context.Context,
	id valueobject.CommandID,
) (*entity.AiCommand, error) {
	row := r.pool.QueryRow(ctx, selectAiCommandSQL, id.Value())
	return scanAiCommand(row)
}

// GetByIDTx retrieves a single command by ID using FOR UPDATE inside a transaction.
func (r *PostgresAiCommandRepository) GetByIDTx(
	ctx context.Context,
	tx repository.Transaction,
	id valueobject.CommandID,
) (*entity.AiCommand, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return nil, err
	}
	row := pgxTx.QueryRow(ctx, selectAiCommandSQL+" FOR UPDATE", id.Value())
	return scanAiCommand(row)
}

// UpdateTx persists changes to an existing ai_command row.
// Mutable fields: status, results, plan_ops (rare), executed_at, cancelled_at,
// llm_tokens_in/out (if re-tried). We re-write everything except id/user_id/input/created_at.
func (r *PostgresAiCommandRepository) UpdateTx(
	ctx context.Context,
	tx repository.Transaction,
	cmd *entity.AiCommand,
) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	planJSON, err := encodePlan(cmd.PlanOps())
	if err != nil {
		return err
	}
	resultsJSON, err := encodeResults(cmd.Results())
	if err != nil {
		return err
	}
	const q = `
		UPDATE ai_commands SET
			plan_ops       = $1,
			explanation    = $2,
			status         = $3,
			llm_model      = $4,
			llm_tokens_in  = $5,
			llm_tokens_out = $6,
			results        = $7,
			expires_at     = $8,
			executed_at    = $9,
			cancelled_at   = $10
		WHERE id = $11`
	tag, err := pgxTx.Exec(ctx, q,
		planJSON,
		cmd.Explanation(),
		cmd.Status().String(),
		cmd.LLMModel(),
		cmd.LLMTokensIn(),
		cmd.LLMTokensOut(),
		resultsJSON,
		cmd.ExpiresAt(),
		cmd.ExecutedAt(),
		cmd.CancelledAt(),
		cmd.ID().Value(),
	)
	if err != nil {
		return fmt.Errorf("update ai_command: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.ErrCommandNotFound
	}
	return nil
}

// ExpirePendingBatch transitions expired awaiting_confirmation rows.
// Uses a self-limiting subquery to avoid full-scan and to allow the janitor
// to call us in a tight loop without deadlocks.
func (r *PostgresAiCommandRepository) ExpirePendingBatch(
	ctx context.Context,
	limit int,
) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	const q = `
		UPDATE ai_commands
		SET status = 'expired'
		WHERE id IN (
			SELECT id FROM ai_commands
			WHERE status = 'awaiting_confirmation'
			  AND expires_at <= NOW()
			ORDER BY expires_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)`
	tag, err := r.pool.Exec(ctx, q, limit)
	if err != nil {
		return 0, fmt.Errorf("expire pending ai_commands: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ----------------------- helpers -----------------------

const selectAiCommandSQL = `
	SELECT
		id, user_id, input, plan_ops, explanation, status,
		llm_model, llm_tokens_in, llm_tokens_out, results,
		created_at, expires_at, executed_at, cancelled_at
	FROM ai_commands
	WHERE id = $1`

func scanAiCommand(row pgx.Row) (*entity.AiCommand, error) {
	var (
		id           uuid.UUID
		userID       uuid.UUID
		input        string
		planJSON     []byte
		explanation  string
		statusStr    string
		llmModel     string
		llmTokensIn  int
		llmTokensOut int
		resultsJSON  []byte
		createdAt    time.Time
		expiresAt    time.Time
		executedAt   *time.Time
		cancelledAt  *time.Time
	)
	err := row.Scan(
		&id, &userID, &input, &planJSON, &explanation, &statusStr,
		&llmModel, &llmTokensIn, &llmTokensOut, &resultsJSON,
		&createdAt, &expiresAt, &executedAt, &cancelledAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, domainerr.ErrCommandNotFound
		}
		return nil, fmt.Errorf("scan ai_command: %w", err)
	}
	ops, err := decodePlan(planJSON)
	if err != nil {
		return nil, err
	}
	results, err := decodeResults(resultsJSON)
	if err != nil {
		return nil, err
	}
	status, err := valueobject.ParseCommandStatus(statusStr)
	if err != nil {
		// Unknown status in DB — surface the parse error instead of crashing.
		return nil, err
	}
	return entity.ReconstructAiCommand(
		valueobject.CommandIDFromUUID(id),
		valueobject.UserIDFromUUID(userID),
		valueobject.PlanInputFromString(input),
		ops,
		explanation,
		status,
		llmModel,
		llmTokensIn,
		llmTokensOut,
		results,
		createdAt,
		expiresAt,
		executedAt,
		cancelledAt,
	), nil
}

func encodePlan(ops []entity.Operation) ([]byte, error) {
	out := make([]jsonOperation, 0, len(ops))
	for _, op := range ops {
		jo := jsonOperation{
			Kind:   op.Kind().String(),
			NodeID: op.NodeID().String(),
		}
		if op.Kind().IsRename() {
			jo.NewName = op.NewName()
		}
		if op.Kind().IsMove() {
			if p := op.NewParentID(); p != nil {
				s := p.String()
				jo.NewParentID = &s
			}
		}
		out = append(out, jo)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal plan_ops: %w", err)
	}
	return b, nil
}

func decodePlan(raw []byte) ([]entity.Operation, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var jsonOps []jsonOperation
	if err := json.Unmarshal(raw, &jsonOps); err != nil {
		return nil, fmt.Errorf("unmarshal plan_ops: %w", err)
	}
	ops := make([]entity.Operation, 0, len(jsonOps))
	for _, jo := range jsonOps {
		kind, err := valueobject.ParseOperationKind(jo.Kind)
		if err != nil {
			return nil, err
		}
		nodeID, err := valueobject.ParseNodeID(jo.NodeID)
		if err != nil {
			return nil, err
		}
		var newParent *valueobject.NodeID
		if jo.NewParentID != nil {
			p, err := valueobject.ParseNodeID(*jo.NewParentID)
			if err != nil {
				return nil, err
			}
			newParent = &p
		}
		ops = append(ops, entity.ReconstructOperation(kind, nodeID, jo.NewName, newParent))
	}
	return ops, nil
}

func encodeResults(rs []entity.OperationResult) (any, error) {
	if len(rs) == 0 {
		return nil, nil
	}
	out := make([]jsonOperationResult, 0, len(rs))
	for _, r := range rs {
		out = append(out, jsonOperationResult{
			Index:        r.Index(),
			Kind:         r.Kind().String(),
			NodeID:       r.NodeID().String(),
			Success:      r.Success(),
			ErrorCode:    r.ErrorCode(),
			ErrorMessage: r.ErrorMessage(),
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal results: %w", err)
	}
	return b, nil
}

func decodeResults(raw []byte) ([]entity.OperationResult, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var jsonRs []jsonOperationResult
	if err := json.Unmarshal(raw, &jsonRs); err != nil {
		return nil, fmt.Errorf("unmarshal results: %w", err)
	}
	out := make([]entity.OperationResult, 0, len(jsonRs))
	for _, jr := range jsonRs {
		kind, err := valueobject.ParseOperationKind(jr.Kind)
		if err != nil {
			return nil, err
		}
		nodeID, err := valueobject.ParseNodeID(jr.NodeID)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.ReconstructOperationResult(
			jr.Index, kind, nodeID, jr.Success, jr.ErrorCode, jr.ErrorMessage,
		))
	}
	return out, nil
}
