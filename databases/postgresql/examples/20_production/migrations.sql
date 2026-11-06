-- Изменения схемы под нагрузкой: что переписывает таблицу, что нет, и как не устроить очередь блокировок
CREATE TABLE users (id bigint PRIMARY KEY, email text, created_at timestamptz NOT NULL DEFAULT '2026-01-01');
INSERT INTO users (id, email) SELECT i, 'u' || i || '@example.com' FROM generate_series(1, 100000) i;

-- Каждая миграция начинается с короткого lock_timeout: лучше упасть и повторить, чем повесить всех
SET lock_timeout = '2s';
SET statement_timeout = '15min';

-- 1. Добавление колонки со значением по умолчанию — мгновенно (PostgreSQL 11+), таблица не переписывается
SELECT relfilenode AS before FROM pg_class WHERE relname = 'users' \gset
ALTER TABLE users ADD COLUMN is_active boolean NOT NULL DEFAULT true;
SELECT relfilenode = :before AS not_rewritten FROM pg_class WHERE relname = 'users';

-- 2. Смена типа int → bigint, text → int и т. п. переписывает таблицу под AccessExclusiveLock
ALTER TABLE users ADD COLUMN score int NOT NULL DEFAULT 0;
SELECT relfilenode AS before FROM pg_class WHERE relname = 'users' \gset
ALTER TABLE users ALTER COLUMN score TYPE bigint;
SELECT relfilenode = :before AS not_rewritten FROM pg_class WHERE relname = 'users';

-- 3. NOT NULL на существующей колонке без долгой блокировки: NOT VALID, затем VALIDATE (PostgreSQL 18)
ALTER TABLE users ADD CONSTRAINT users_email_not_null NOT NULL email NOT VALID;   -- быстро: новые строки уже проверяются
INSERT INTO users (id, email) VALUES (100001, NULL);
ALTER TABLE users VALIDATE CONSTRAINT users_email_not_null;    -- проверка старых строк без блокировки записи
\d users

-- 4. Внешний ключ: NOT VALID + VALIDATE по той же схеме
CREATE TABLE sessions (id bigint PRIMARY KEY, user_id bigint NOT NULL);
INSERT INTO sessions SELECT i, i FROM generate_series(1, 1000) i;
ALTER TABLE sessions ADD CONSTRAINT sessions_user_fk FOREIGN KEY (user_id) REFERENCES users NOT VALID;
ALTER TABLE sessions VALIDATE CONSTRAINT sessions_user_fk;

-- 5. Индекс без блокировки записи (нельзя внутри транзакции)
CREATE INDEX CONCURRENTLY users_email_idx ON users (email);
BEGIN;
CREATE INDEX CONCURRENTLY users_created_idx ON users (created_at);
ROLLBACK;
SELECT indexrelid::regclass AS index, indisvalid FROM pg_index WHERE indrelid = 'users'::regclass ORDER BY 1;

-- 6. Массовое обновление данных — порциями, а не одним UPDATE на всю таблицу
UPDATE users SET score = 1 WHERE id IN (SELECT id FROM users WHERE score = 0 ORDER BY id LIMIT 10000);
SELECT count(*) FILTER (WHERE score = 1) AS updated FROM users;

-- 7. Повтор миграции при lock_timeout: шаблон
DO $$
DECLARE attempt int := 0;
BEGIN
    LOOP
        BEGIN
            SET LOCAL lock_timeout = '1s';
            ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname text;
            EXIT;
        EXCEPTION WHEN lock_not_available THEN
            attempt := attempt + 1;
            IF attempt >= 5 THEN RAISE; END IF;
            PERFORM pg_sleep(attempt);          -- пауза растёт с каждой попыткой
        END;
    END LOOP;
END $$;
SELECT count(*) AS has_nickname FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'nickname';
