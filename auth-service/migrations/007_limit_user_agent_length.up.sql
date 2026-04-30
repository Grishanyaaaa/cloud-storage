-- Limit user_agent length to prevent DoS attacks via huge User-Agent headers
-- Change from TEXT to VARCHAR(1024) in both audit_logs and refresh_tokens tables

ALTER TABLE audit_logs ALTER COLUMN user_agent TYPE VARCHAR(1024);
ALTER TABLE refresh_tokens ALTER COLUMN user_agent TYPE VARCHAR(1024);
