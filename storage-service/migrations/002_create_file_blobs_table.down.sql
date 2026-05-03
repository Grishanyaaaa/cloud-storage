DROP TRIGGER IF EXISTS trg_file_blobs_set_updated_at ON file_blobs;
DROP TABLE IF EXISTS file_blobs CASCADE;
DROP TYPE IF EXISTS blob_status;
