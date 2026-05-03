CREATE TABLE user_roots (
    user_id    UUID PRIMARY KEY,
    root_id    UUID NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One root node per user is also enforced via uq_nodes_owner_root_alive,
-- but we additionally guarantee the binding is unique on the root_id side.
CREATE UNIQUE INDEX uq_user_roots_root_id ON user_roots (root_id);
