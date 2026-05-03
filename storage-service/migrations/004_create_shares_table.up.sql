CREATE TYPE share_perm AS ENUM ('view', 'edit');

CREATE TABLE shares (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id     UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    owner_id    UUID NOT NULL,
    token_hash  VARCHAR(64) NOT NULL,                                      -- sha256(token), hex
    permission  share_perm NOT NULL,
    expires_at  TIMESTAMPTZ NULL,
    revoked_at  TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Token hash is globally unique (per spec — one row per generated token).
CREATE UNIQUE INDEX uq_shares_token_hash ON shares (token_hash);

-- Owner listing: alive shares only.
CREATE INDEX idx_shares_owner_alive
    ON shares (owner_id)
    WHERE revoked_at IS NULL;

-- Listing shares of a node.
CREATE INDEX idx_shares_node_alive
    ON shares (node_id)
    WHERE revoked_at IS NULL;

-- Janitor: shares with expires_at past now.
CREATE INDEX idx_shares_expires_alive
    ON shares (expires_at)
    WHERE revoked_at IS NULL AND expires_at IS NOT NULL;
