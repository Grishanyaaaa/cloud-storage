-- Remove email format CHECK constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_email_format;
