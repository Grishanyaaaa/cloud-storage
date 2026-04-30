-- Revert to separate indexes
DROP INDEX IF EXISTS idx_audit_logs_user_id_created_at;

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
