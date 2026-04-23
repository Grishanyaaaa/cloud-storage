CREATE TABLE refresh_tokens (
                                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token_hash VARCHAR(255) NOT NULL,
                                expires_at TIMESTAMP NOT NULL,
                                created_at TIMESTAMP DEFAULT NOW(),
                                revoked_at TIMESTAMP NULL,
                                ip_address INET NULL,
                                user_agent TEXT NULL
);

-- Поиск по хешу токена основной lookup при refresh
CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Все активные токены юзера нужно для revoke all при logout
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id) WHERE revoked_at IS NULL;

-- Чистка просроченных токенов по крону
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);