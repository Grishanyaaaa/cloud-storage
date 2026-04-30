-- Add composite index for efficient user audit log queries
-- Optimizes FindByUserID query: WHERE user_id = $1 ORDER BY created_at DESC
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_created_at;

CREATE INDEX idx_audit_logs_user_id_created_at ON audit_logs(user_id, created_at DESC);
