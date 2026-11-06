-- Таблица на 200 000 заказов с предсказуемыми данными
SET max_parallel_workers_per_gather = 0;   -- планы без параллельности: проще читать
CREATE TABLE orders (
    id          bigint PRIMARY KEY,
    customer_id int NOT NULL,
    status      text NOT NULL,
    email       text NOT NULL,
    total       int NOT NULL,
    created_at  timestamptz NOT NULL
);
INSERT INTO orders
SELECT i,
       i % 20000,
       CASE WHEN i % 100 = 0 THEN 'new' WHEN i % 10 = 0 THEN 'cancelled' ELSE 'done' END,
       'User' || (i % 20000) || '@Example.com',
       (i * 7919) % 10000,
       '2026-01-01'::timestamptz + (i || ' minutes')::interval
FROM generate_series(1, 200000) i;
ANALYZE orders;

-- Без индекса: последовательное чтение всей таблицы
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM orders WHERE customer_id = 42;

CREATE INDEX orders_customer_idx ON orders (customer_id);
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM orders WHERE customer_id = 42;

-- Составной индекс: равенство первым, диапазон или сортировка — следом
CREATE INDEX orders_customer_created_idx ON orders (customer_id, created_at);
EXPLAIN (COSTS OFF)
SELECT id, created_at FROM orders WHERE customer_id = 42 ORDER BY created_at DESC LIMIT 3;

-- Skip scan (PostgreSQL 18): индекс используется даже без условия на первую колонку,
-- если у неё мало разных значений
CREATE INDEX orders_status_created_idx ON orders (status, created_at);
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT count(*) FROM orders WHERE created_at BETWEEN '2026-03-01' AND '2026-03-01 01:00';

-- Частичный индекс: только «новые» заказы — 1% таблицы
CREATE INDEX orders_new_idx ON orders (created_at) WHERE status = 'new';
EXPLAIN (COSTS OFF) SELECT id FROM orders WHERE status = 'new' ORDER BY created_at LIMIT 10;

-- Индекс по выражению: поиск email без учёта регистра
EXPLAIN (COSTS OFF) SELECT id FROM orders WHERE lower(email) = 'user42@example.com' LIMIT 1;
CREATE INDEX orders_email_lower_idx ON orders (lower(email));
EXPLAIN (COSTS OFF) SELECT id FROM orders WHERE lower(email) = 'user42@example.com' LIMIT 1;

-- Покрывающий индекс и Index Only Scan: все нужные колонки есть в индексе
CREATE INDEX orders_cust_total_idx ON orders (customer_id) INCLUDE (total);
VACUUM orders;    -- карта видимости нужна, чтобы не ходить в таблицу
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT sum(total) FROM orders WHERE customer_id = 42;

-- Функция над колонкой «ломает» обычный индекс
EXPLAIN (COSTS OFF) SELECT count(*) FROM orders WHERE created_at::date = '2026-03-01';
EXPLAIN (COSTS OFF) SELECT count(*) FROM orders
WHERE created_at >= '2026-03-01' AND created_at < '2026-03-02';

-- BRIN: крошечный индекс для данных, упорядоченных физически (время вставки)
CREATE INDEX orders_created_brin ON orders USING brin (created_at);
SELECT relname, pg_size_pretty(pg_relation_size(oid)) AS size
FROM pg_class
WHERE relname IN ('orders', 'orders_pkey', 'orders_status_created_idx', 'orders_new_idx', 'orders_created_brin')
ORDER BY pg_relation_size(oid) DESC;

-- GIN + pg_trgm: поиск подстроки LIKE '%...%'
CREATE EXTENSION pg_trgm;
CREATE INDEX orders_email_trgm ON orders USING gin (email gin_trgm_ops);
EXPLAIN (COSTS OFF) SELECT count(*) FROM orders WHERE email LIKE '%er1999@%';

-- Какие индексы есть и какого они размера
SELECT indexrelname AS index, pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes WHERE relname = 'orders' ORDER BY pg_relation_size(indexrelid) DESC;
