-- Revert user_agent length limit
ALTER TABLE audit_logs ALTER COLUMN user_agent TYPE TEXT;
ALTER TABLE refresh_tokens ALTER COLUMN user_agent TYPE TEXT;
