CREATE TYPE blob_status AS ENUM ('pending', 'active', 'failed');

CREATE TABLE file_blobs (
    node_id      UUID PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    storage_key  VARCHAR(1024) NOT NULL,
    mime_type    VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    checksum     VARCHAR(64) NOT NULL DEFAULT '',
    status       blob_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NULL
);

-- Janitor index: pending blobs whose pre-signed URL is past TTL.
CREATE INDEX idx_file_blobs_pending_expired
    ON file_blobs (expires_at)
    WHERE status = 'pending' AND expires_at IS NOT NULL;

-- Storage key uniqueness — one blob per key.
CREATE UNIQUE INDEX uq_file_blobs_storage_key
    ON file_blobs (storage_key);

CREATE TRIGGER trg_file_blobs_set_updated_at
BEFORE UPDATE ON file_blobs
FOR EACH ROW EXECUTE FUNCTION nodes_set_updated_at();
