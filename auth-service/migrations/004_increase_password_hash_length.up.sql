-- Увеличиваем размер колонки password_hash для поддержки различных алгоритмов хеширования
ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(255);
