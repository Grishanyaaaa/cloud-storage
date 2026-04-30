-- Откатываем к старому индексу
DROP INDEX IF EXISTS idx_refresh_tokens_cleanup;

-- Восстанавливаем оригинальный индекс
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
