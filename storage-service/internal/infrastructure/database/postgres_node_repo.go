package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

const (
	nodesNameUniqueIdx = "uq_nodes_parent_name_alive"
	nodesRootUniqueIdx = "uq_nodes_owner_root_alive"
)

// Compile-time check
var _ repository.NodeRepository = (*NodeRepositoryPg)(nil)

// NodeRepositoryPg is the PostgreSQL implementation of repository.NodeRepository.
type NodeRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewNodeRepository(pool *pgxpool.Pool) *NodeRepositoryPg {
	return &NodeRepositoryPg{pool: pool}
}

const nodeColumns = `id, owner_id, parent_id, kind, name, path, depth, created_at, updated_at, deleted_at`

func (r *NodeRepositoryPg) Create(ctx context.Context, n *entity.Node) error {
	return r.insert(ctx, r.pool, n)
}

func (r *NodeRepositoryPg) CreateTx(ctx context.Context, tx repository.Transaction, n *entity.Node) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	return r.insert(ctx, pgxTx, n)
}

// queryRow / execSQL / querySQL are small helpers to avoid duplicating
// pool vs. tx code paths inside repository implementations.
func queryRow(ctx context.Context, x any, sql string, args ...any) pgx.Row {
	switch q := x.(type) {
	case *pgxpool.Pool:
		return q.QueryRow(ctx, sql, args...)
	case pgx.Tx:
		return q.QueryRow(ctx, sql, args...)
	default:
		panic(fmt.Sprintf("queryRow: unsupported type %T", x))
	}
}

func execSQL(ctx context.Context, x any, sql string, args ...any) (int64, error) {
	switch q := x.(type) {
	case *pgxpool.Pool:
		ct, err := q.Exec(ctx, sql, args...)
		if err != nil {
			return 0, err
		}
		return ct.RowsAffected(), nil
	case pgx.Tx:
		ct, err := q.Exec(ctx, sql, args...)
		if err != nil {
			return 0, err
		}
		return ct.RowsAffected(), nil
	default:
		panic(fmt.Sprintf("execSQL: unsupported type %T", x))
	}
}

func querySQL(ctx context.Context, x any, sql string, args ...any) (pgx.Rows, error) {
	switch q := x.(type) {
	case *pgxpool.Pool:
		return q.Query(ctx, sql, args...)
	case pgx.Tx:
		return q.Query(ctx, sql, args...)
	default:
		panic(fmt.Sprintf("querySQL: unsupported type %T", x))
	}
}

func (r *NodeRepositoryPg) insert(ctx context.Context, x any, n *entity.Node) error {
	const q = `
INSERT INTO nodes (id, owner_id, parent_id, kind, name, path, depth, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	var parentID any
	if pid := n.ParentID(); pid != nil {
		parentID = pid.Value()
	} else {
		parentID = nil
	}
	_, err := execSQL(ctx, x, q,
		n.ID().Value(),
		n.OwnerID().Value(),
		parentID,
		string(n.Kind()),
		n.Name().String(),
		n.Path().String(),
		n.Depth(),
		n.CreatedAt(),
		n.UpdatedAt(),
		nullableTime(n.DeletedAt()),
	)
	if err != nil {
		if isUniqueViolationOnConstraint(err, nodesNameUniqueIdx) {
			return domainerr.ErrNodeNameTaken
		}
		if isUniqueViolationOnConstraint(err, nodesRootUniqueIdx) {
			return domainerr.ErrUserRootAlreadyExists
		}
		return fmt.Errorf("insert node: %w", err)
	}
	return nil
}

func (r *NodeRepositoryPg) GetByID(ctx context.Context, id valueobject.NodeID) (*entity.Node, error) {
	const q = `SELECT ` + nodeColumns + ` FROM nodes WHERE id = $1`
	return r.scanRow(ctx, queryRow(ctx, r.pool, q, id.Value()))
}

func (r *NodeRepositoryPg) GetByIDForOwner(ctx context.Context, owner valueobject.UserID, id valueobject.NodeID) (*entity.Node, error) {
	const q = `SELECT ` + nodeColumns + ` FROM nodes WHERE id = $1 AND owner_id = $2`
	return r.scanRow(ctx, queryRow(ctx, r.pool, q, id.Value(), owner.Value()))
}

func (r *NodeRepositoryPg) GetRootByOwner(ctx context.Context, owner valueobject.UserID) (*entity.Node, error) {
	const q = `SELECT ` + nodeColumns + ` FROM nodes
WHERE owner_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
LIMIT 1`
	n, err := r.scanRow(ctx, queryRow(ctx, r.pool, q, owner.Value()))
	if err != nil && err == domainerr.ErrNodeNotFound {
		return nil, domainerr.ErrUserRootNotFound
	}
	return n, err
}

func (r *NodeRepositoryPg) ListChildren(ctx context.Context, owner valueobject.UserID, parentID valueobject.NodeID, f repository.NodeFilter) ([]*entity.Node, string, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	cur, err := decodeNodeCursor(f.Cursor)
	if err != nil {
		return nil, "", err
	}
	args := []any{owner.Value(), parentID.Value(), limit + 1}
	where := []string{"owner_id = $1", "parent_id = $2"}
	if !f.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if cur != nil {
		args = append(args, cur.Name, cur.ID)
		where = append(where, fmt.Sprintf("(name, id) > ($%d, $%d)", len(args)-1, len(args)))
	}
	q := fmt.Sprintf(`SELECT %s FROM nodes WHERE %s ORDER BY name ASC, id ASC LIMIT $3`,
		nodeColumns, strings.Join(where, " AND "))
	rows, err := querySQL(ctx, r.pool, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query children: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.Node, 0, limit)
	for rows.Next() {
		n, err := r.scanRowFromRows(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(out) > limit {
		last := out[limit-1]
		next = encodeNodeCursor(last.Name().String(), last.ID().Value())
		out = out[:limit]
	}
	return out, next, nil
}

func (r *NodeRepositoryPg) ListSubtree(ctx context.Context, owner valueobject.UserID, root *entity.Node, maxDepth int, includeDeleted bool) ([]*entity.Node, error) {
	args := []any{
		owner.Value(),
		root.Path().String(),
		root.Path().LikePrefix(),
	}
	where := []string{"owner_id = $1", "(path = $2 OR path LIKE $3)"}
	if maxDepth > 0 {
		args = append(args, root.Depth()+maxDepth)
		where = append(where, fmt.Sprintf("depth <= $%d", len(args)))
	}
	if !includeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	q := fmt.Sprintf(`SELECT %s FROM nodes WHERE %s ORDER BY depth ASC, name ASC, id ASC`,
		nodeColumns, strings.Join(where, " AND "))
	rows, err := querySQL(ctx, r.pool, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query subtree: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.Node, 0)
	for rows.Next() {
		n, err := r.scanRowFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NodeRepositoryPg) Rename(ctx context.Context, n *entity.Node) error {
	const q = `UPDATE nodes SET name = $2, updated_at = $3 WHERE id = $1`
	rows, err := execSQL(ctx, r.pool, q, n.ID().Value(), n.Name().String(), n.UpdatedAt())
	if err != nil {
		if isUniqueViolationOnConstraint(err, nodesNameUniqueIdx) {
			return domainerr.ErrNodeNameTaken
		}
		return fmt.Errorf("rename node: %w", err)
	}
	if rows == 0 {
		return domainerr.ErrNodeNotFound
	}
	return nil
}

func (r *NodeRepositoryPg) MoveTx(ctx context.Context, tx repository.Transaction, oldPath valueobject.NodePath, n *entity.Node) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	// Update the moved node first.
	var parentID any
	if pid := n.ParentID(); pid != nil {
		parentID = pid.Value()
	}
	const updNode = `UPDATE nodes
SET parent_id = $2, path = $3, depth = $4, updated_at = $5
WHERE id = $1`
	if _, err := pgxTx.Exec(ctx, updNode,
		n.ID().Value(), parentID, n.Path().String(), n.Depth(), n.UpdatedAt(),
	); err != nil {
		if isUniqueViolationOnConstraint(err, nodesNameUniqueIdx) {
			return domainerr.ErrNodeNameTaken
		}
		return fmt.Errorf("update moved node: %w", err)
	}
	// Rewrite path & depth for every descendant. We replace the old prefix with the new prefix.
	// path = new || substring(path FROM length(old)+1)
	// depth = depth + delta
	delta := n.Depth() - oldPath.Depth()
	const updDescendants = `UPDATE nodes
SET path = $1 || substring(path FROM $4),
    depth = depth + $5,
    updated_at = $6
WHERE path LIKE $2 AND id <> $3`
	likePrefix := oldPath.LikePrefix()
	from := len(oldPath.String()) + 1
	if _, err := pgxTx.Exec(ctx, updDescendants,
		n.Path().String(), likePrefix, n.ID().Value(), from, delta, n.UpdatedAt(),
	); err != nil {
		return fmt.Errorf("rewrite descendant paths: %w", err)
	}
	return nil
}

func (r *NodeRepositoryPg) SoftDeleteSubtreeTx(ctx context.Context, tx repository.Transaction, root *entity.Node, now time.Time) (int64, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return 0, err
	}
	const q = `UPDATE nodes
SET deleted_at = $1, updated_at = $1
WHERE (path = $2 OR path LIKE $3) AND deleted_at IS NULL`
	ct, err := pgxTx.Exec(ctx, q, now, root.Path().String(), root.Path().LikePrefix())
	if err != nil {
		return 0, fmt.Errorf("soft delete subtree: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *NodeRepositoryPg) RestoreSubtreeTx(ctx context.Context, tx repository.Transaction, root *entity.Node, now time.Time) (int64, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return 0, err
	}
	const q = `UPDATE nodes
SET deleted_at = NULL, updated_at = $1
WHERE (path = $2 OR path LIKE $3) AND deleted_at IS NOT NULL`
	ct, err := pgxTx.Exec(ctx, q, now, root.Path().String(), root.Path().LikePrefix())
	if err != nil {
		return 0, fmt.Errorf("restore subtree: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *NodeRepositoryPg) scanRow(ctx context.Context, row pgx.Row) (*entity.Node, error) {
	var (
		id, owner uuid.UUID
		parent    *uuid.UUID
		kind      string
		name      string
		path      string
		depth     int
		createdAt time.Time
		updatedAt time.Time
		deletedAt *time.Time
	)
	if err := row.Scan(&id, &owner, &parent, &kind, &name, &path, &depth, &createdAt, &updatedAt, &deletedAt); err != nil {
		if isNoRows(err) {
			return nil, domainerr.ErrNodeNotFound
		}
		return nil, fmt.Errorf("scan node: %w", err)
	}
	return buildNode(id, owner, parent, kind, name, path, depth, createdAt, updatedAt, deletedAt)
}

func (r *NodeRepositoryPg) scanRowFromRows(rows pgx.Rows) (*entity.Node, error) {
	var (
		id, owner uuid.UUID
		parent    *uuid.UUID
		kind      string
		name      string
		path      string
		depth     int
		createdAt time.Time
		updatedAt time.Time
		deletedAt *time.Time
	)
	if err := rows.Scan(&id, &owner, &parent, &kind, &name, &path, &depth, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, fmt.Errorf("scan node row: %w", err)
	}
	return buildNode(id, owner, parent, kind, name, path, depth, createdAt, updatedAt, deletedAt)
}

func buildNode(
	id, owner uuid.UUID,
	parent *uuid.UUID,
	kind, name, path string,
	depth int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (*entity.Node, error) {
	k, err := valueobject.ParseNodeKind(kind)
	if err != nil {
		return nil, err
	}
	var pid *valueobject.NodeID
	if parent != nil {
		v := valueobject.NodeIDFromUUID(*parent)
		pid = &v
	}
	return entity.ReconstructNode(
		valueobject.NodeIDFromUUID(id),
		valueobject.UserIDFromUUID(owner),
		pid,
		k,
		valueobject.NodeNameFromTrusted(name),
		valueobject.NodePathFromTrusted(path),
		depth,
		createdAt,
		updatedAt,
		deletedAt,
	), nil
}

// ----------------- cursor encoding for ListChildren --------------------

type nodeCursor struct {
	Name string
	ID   uuid.UUID
}

func encodeNodeCursor(name string, id uuid.UUID) string {
	raw := name + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeNodeCursor(s string) (*nodeCursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor id: %w", err)
	}
	return &nodeCursor{Name: parts[0], ID: id}, nil
}
