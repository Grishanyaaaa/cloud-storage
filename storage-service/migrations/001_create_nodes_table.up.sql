CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE node_kind AS ENUM ('folder', 'file');

CREATE TABLE nodes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL,
    parent_id   UUID NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    kind        node_kind NOT NULL,
    name        VARCHAR(255) NOT NULL CHECK (char_length(name) > 0),
    path        TEXT NOT NULL CHECK (char_length(path) <= 4096),
    depth       INT NOT NULL CHECK (depth >= 1),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ NULL,

    CONSTRAINT nodes_root_has_no_parent CHECK (
        (parent_id IS NULL AND depth = 1) OR (parent_id IS NOT NULL AND depth > 1)
    )
);

-- One alive name per parent: prevents duplicates only among non-deleted siblings.
CREATE UNIQUE INDEX uq_nodes_parent_name_alive
    ON nodes (parent_id, name)
    WHERE deleted_at IS NULL AND parent_id IS NOT NULL;

-- One alive root per owner.
CREATE UNIQUE INDEX uq_nodes_owner_root_alive
    ON nodes (owner_id)
    WHERE parent_id IS NULL AND deleted_at IS NULL;

-- Listing children efficiently.
CREATE INDEX idx_nodes_parent_alive
    ON nodes (parent_id)
    WHERE deleted_at IS NULL;

-- Subtree queries via path prefix.
CREATE INDEX idx_nodes_path_prefix
    ON nodes (path text_pattern_ops);

-- Owner scoping.
CREATE INDEX idx_nodes_owner_alive
    ON nodes (owner_id)
    WHERE deleted_at IS NULL;

-- Trigger: keep updated_at fresh on every UPDATE.
CREATE OR REPLACE FUNCTION nodes_set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_nodes_set_updated_at
BEFORE UPDATE ON nodes
FOR EACH ROW EXECUTE FUNCTION nodes_set_updated_at();
