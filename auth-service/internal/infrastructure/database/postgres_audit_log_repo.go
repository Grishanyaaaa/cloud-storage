package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Compile-time check: AuditLogRepositoryPg implements repository.AuditLogRepository
var _ repository.AuditLogRepository = (*AuditLogRepositoryPg)(nil)

type AuditLogRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepositoryPg {
	return &AuditLogRepositoryPg{pool: pool}
}

func (r *AuditLogRepositoryPg) Save(ctx context.Context, log *entity.AuditLog) error {
	const q = `
		INSERT INTO audit_logs (id, user_id, action, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, q,
		log.ID().String(),
		log.UserID().String(),
		string(log.Action()),
		nullableString(log.IPAddress()),
		nullableString(log.UserAgent()),
		log.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

func (r *AuditLogRepositoryPg) GetByID(ctx context.Context, id valueobject.AuditLogID) (*entity.AuditLog, error) {
	const q = `
		SELECT id, user_id, action, ip_address, user_agent, created_at
		FROM audit_logs WHERE id = $1`

	return r.scanLog(ctx, q, id.String())
}

func (r *AuditLogRepositoryPg) FindByUserID(ctx context.Context, userID valueobject.UserID, limit, offset int) ([]*entity.AuditLog, error) {
	const q = `
		SELECT id, user_id, action, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*entity.AuditLog
	for rows.Next() {
		log, err := r.scanFromRow(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}

	return logs, nil
}

func (r *AuditLogRepositoryPg) scanLog(ctx context.Context, query string, args ...any) (*entity.AuditLog, error) {
	row := r.pool.QueryRow(ctx, query, args...)

	var (
		id        string
		userID    string
		action    string
		ipAddress *string
		userAgent *string
		createdAt time.Time
	)

	err := row.Scan(&id, &userID, &action, &ipAddress, &userAgent, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.ErrAuditLogNotFound
		}
		return nil, fmt.Errorf("scan audit log: %w", err)
	}

	return r.toEntity(id, userID, action, ipAddress, userAgent, createdAt)
}

func (r *AuditLogRepositoryPg) scanFromRow(rows pgx.Rows) (*entity.AuditLog, error) {
	var (
		id        string
		userID    string
		action    string
		ipAddress *string
		userAgent *string
		createdAt time.Time
	)

	if err := rows.Scan(&id, &userID, &action, &ipAddress, &userAgent, &createdAt); err != nil {
		return nil, fmt.Errorf("scan audit log row: %w", err)
	}

	return r.toEntity(id, userID, action, ipAddress, userAgent, createdAt)
}

// toEntity — маппинг сырых данных в доменную сущность, вынесен чтобы не дублировать
func (r *AuditLogRepositoryPg) toEntity(
	id, userID, action string,
	ipAddress, userAgent *string,
	createdAt time.Time,
) (*entity.AuditLog, error) {
	logID, err := valueobject.ParseAuditLogID(id)
	if err != nil {
		return nil, fmt.Errorf("parse audit log id: %w", err)
	}

	uid, err := valueobject.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	return entity.ReconstructAuditLog(
		logID, uid,
		entity.Action(action),
		derefString(ipAddress),
		derefString(userAgent),
		createdAt,
	), nil
}

// хелперы для nullable полей — ip_address и user_agent могут быть NULL
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
