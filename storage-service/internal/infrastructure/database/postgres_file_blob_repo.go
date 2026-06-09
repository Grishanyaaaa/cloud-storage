package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Compile-time check
var _ repository.FileBlobRepository = (*FileBlobRepositoryPg)(nil)

type FileBlobRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewFileBlobRepository(pool *pgxpool.Pool) *FileBlobRepositoryPg {
	return &FileBlobRepositoryPg{pool: pool}
}

const fileBlobColumns = `node_id, storage_key, mime_type, size_bytes, checksum, status, created_at, updated_at, expires_at`

func (r *FileBlobRepositoryPg) Create(ctx context.Context, b *entity.FileBlob) error {
	return r.insert(ctx, r.pool, b)
}

func (r *FileBlobRepositoryPg) CreateTx(ctx context.Context, tx repository.Transaction, b *entity.FileBlob) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	return r.insert(ctx, pgxTx, b)
}

func (r *FileBlobRepositoryPg) insert(ctx context.Context, x any, b *entity.FileBlob) error {
	const q = `
INSERT INTO file_blobs (node_id, storage_key, mime_type, size_bytes, checksum, status, created_at, updated_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := execSQL(ctx, x, q,
		b.NodeID().Value(),
		b.StorageKey().String(),
		b.MimeType().String(),
		b.Size().Value(),
		b.Checksum(),
		string(b.Status()),
		b.CreatedAt(),
		b.UpdatedAt(),
		nullableTime(b.ExpiresAt()),
	)
	if err != nil {
		return fmt.Errorf("insert file blob: %w", err)
	}
	return nil
}

func (r *FileBlobRepositoryPg) GetByNodeID(ctx context.Context, nodeID valueobject.NodeID) (*entity.FileBlob, error) {
	const q = `SELECT ` + fileBlobColumns + ` FROM file_blobs WHERE node_id = $1`
	row := queryRow(ctx, r.pool, q, nodeID.Value())
	return r.scanRow(row)
}

func (r *FileBlobRepositoryPg) GetByNodeIDs(ctx context.Context, nodeIDs []valueobject.NodeID) (map[valueobject.NodeID]*entity.FileBlob, error) {
	if len(nodeIDs) == 0 {
		return make(map[valueobject.NodeID]*entity.FileBlob), nil
	}
	ids := make([]uuid.UUID, len(nodeIDs))
	for i, nid := range nodeIDs {
		ids[i] = nid.Value()
	}
	const q = `SELECT ` + fileBlobColumns + ` FROM file_blobs WHERE node_id = ANY($1)`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("get blobs by node ids: %w", err)
	}
	defer rows.Close()

	result := make(map[valueobject.NodeID]*entity.FileBlob, len(nodeIDs))
	for rows.Next() {
		blob, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		result[blob.NodeID()] = blob
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blobs: %w", err)
	}
	return result, nil
}

func (r *FileBlobRepositoryPg) Update(ctx context.Context, b *entity.FileBlob) error {
	return r.update(ctx, r.pool, b)
}

func (r *FileBlobRepositoryPg) UpdateTx(ctx context.Context, tx repository.Transaction, b *entity.FileBlob) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	return r.update(ctx, pgxTx, b)
}

func (r *FileBlobRepositoryPg) update(ctx context.Context, x any, b *entity.FileBlob) error {
	const q = `
UPDATE file_blobs
SET storage_key = $2, mime_type = $3, size_bytes = $4, checksum = $5,
    status = $6, expires_at = $7, updated_at = $8
WHERE node_id = $1`
	rows, err := execSQL(ctx, x, q,
		b.NodeID().Value(),
		b.StorageKey().String(),
		b.MimeType().String(),
		b.Size().Value(),
		b.Checksum(),
		string(b.Status()),
		nullableTime(b.ExpiresAt()),
		b.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("update file blob: %w", err)
	}
	if rows == 0 {
		return domainerr.ErrFileBlobNotFound
	}
	return nil
}

func (r *FileBlobRepositoryPg) FailExpiredPending(ctx context.Context, now time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	const q = `
UPDATE file_blobs
SET status = 'failed', updated_at = $1, expires_at = NULL
WHERE node_id IN (
    SELECT node_id FROM file_blobs
    WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at < $1
    LIMIT $2
)`
	ct, err := r.pool.Exec(ctx, q, now, limit)
	if err != nil {
		return 0, fmt.Errorf("fail expired pending: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *FileBlobRepositoryPg) scanRow(row pgx.Row) (*entity.FileBlob, error) {
	var (
		nodeID    uuid.UUID
		key       string
		mime      string
		size      int64
		checksum  string
		status    string
		createdAt time.Time
		updatedAt time.Time
		expiresAt *time.Time
	)
	if err := row.Scan(&nodeID, &key, &mime, &size, &checksum, &status, &createdAt, &updatedAt, &expiresAt); err != nil {
		if isNoRows(err) {
			return nil, domainerr.ErrFileBlobNotFound
		}
		return nil, fmt.Errorf("scan file blob: %w", err)
	}
	return entity.ReconstructFileBlob(
		valueobject.NodeIDFromUUID(nodeID),
		valueobject.StorageKeyFromTrusted(key),
		valueobject.MimeTypeFromTrusted(mime),
		valueobject.SizeBytesFromTrusted(size),
		checksum,
		entity.BlobStatus(status),
		createdAt,
		updatedAt,
		expiresAt,
	), nil
}
