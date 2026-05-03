DROP TRIGGER IF EXISTS trg_nodes_set_updated_at ON nodes;
DROP FUNCTION IF EXISTS nodes_set_updated_at();
DROP TABLE IF EXISTS nodes CASCADE;
DROP TYPE IF EXISTS node_kind;
