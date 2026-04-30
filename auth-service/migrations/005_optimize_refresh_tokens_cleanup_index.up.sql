-- Удаляем старый индекс для cleanup
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;

-- Создаем оптимизированный composite индекс для cleanup операций
-- Этот индекс эффективен для запросов вида: DELETE FROM refresh_tokens WHERE expires_at < $1
-- Также помогает при поиске истекших токенов с учетом статуса revoked_at
CREATE INDEX idx_refresh_tokens_cleanup ON refresh_tokens(expires_at, revoked_at);
