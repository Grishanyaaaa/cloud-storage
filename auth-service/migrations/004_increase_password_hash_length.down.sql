-- Возвращаем обратно к VARCHAR(72) если нужен rollback
ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(72);
