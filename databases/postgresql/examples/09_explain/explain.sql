SET max_parallel_workers_per_gather = 0;
CREATE TABLE customers (id int PRIMARY KEY, city text NOT NULL, region text NOT NULL);
CREATE TABLE orders (id int PRIMARY KEY, customer_id int NOT NULL REFERENCES customers, total int NOT NULL);
-- 30 000 клиентов: ANALYZE читает их целиком, оценки детерминированы.
-- Город однозначно определяет регион: колонки коррелированы.
INSERT INTO customers
SELECT i, 'city' || (i % 100), 'region' || (i % 100 / 10) FROM generate_series(1, 30000) i;
INSERT INTO orders SELECT i, 1 + i % 30000, i % 1000 FROM generate_series(1, 90000) i;
CREATE INDEX orders_customer_idx ON orders (customer_id);
ANALYZE customers, orders;

-- Стоимость: (запуск..всего) в условных единицах, rows — оценка строк, width — средний размер строки
EXPLAIN SELECT * FROM customers WHERE id <= 100;

-- EXPLAIN ANALYZE выполняет запрос и показывает фактические числа рядом с оценками
EXPLAIN (ANALYZE, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM customers WHERE city = 'city1';

-- Ошибка оценки: планировщик считает условия независимыми и перемножает селективности
EXPLAIN (ANALYZE, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM customers WHERE city = 'city1' AND region = 'region0';

-- Расширенная статистика о функциональной зависимости исправляет оценку
CREATE STATISTICS customers_city_region (dependencies) ON city, region FROM customers;
ANALYZE customers;
EXPLAIN (ANALYZE, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM customers WHERE city = 'city1' AND region = 'region0';

-- Что планировщик знает о колонке
SELECT attname, null_frac, n_distinct, (most_common_vals::text::text[])[1:3] AS top3
FROM pg_stats WHERE tablename = 'customers' AND attname IN ('city', 'region')
ORDER BY attname;

-- Три способа соединения
-- Nested Loop: мало строк снаружи, индекс внутри
EXPLAIN (COSTS OFF)
SELECT c.city, o.total FROM customers c JOIN orders o ON o.customer_id = c.id WHERE c.id = 42;

-- Hash Join: большие наборы без подходящего порядка
EXPLAIN (COSTS OFF)
SELECT c.region, sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id GROUP BY c.region;

-- Merge Join: оба входа уже отсортированы по ключу соединения
EXPLAIN (COSTS OFF)
SELECT c.id, o.id FROM customers c JOIN orders o ON o.customer_id = c.id ORDER BY c.id;

-- Сортировка и work_mem: на диске или в памяти
SET work_mem = '64kB';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM orders ORDER BY total, id;
SET work_mem = '16MB';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM orders ORDER BY total, id;
RESET work_mem;

-- Top-N heapsort: LIMIT позволяет не сортировать всё
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM orders ORDER BY total DESC LIMIT 5;

-- EXPLAIN ANALYZE на изменяющем запросе действительно его выполняет — оборачивай в транзакцию
BEGIN;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
DELETE FROM orders WHERE total = 0;
ROLLBACK;
SELECT count(*) FROM orders WHERE total = 0;
