CREATE EXTENSION pg_stat_statements;
SELECT pg_stat_statements_reset() IS NOT NULL AS stats_reset;

CREATE TABLE orders (id int PRIMARY KEY, customer_id int NOT NULL, status text NOT NULL);
INSERT INTO orders SELECT i, i % 100, CASE WHEN i % 10 = 0 THEN 'new' ELSE 'done' END
FROM generate_series(1, 10000) i;
CREATE INDEX orders_status_idx ON orders (status);   -- ни разу не понадобится
ANALYZE orders;

-- Нагрузка: один и тот же запрос с разными параметрами нормализуется в одну строку статистики
SELECT * FROM orders WHERE id = 1;
SELECT * FROM orders WHERE id = 2;
SELECT * FROM orders WHERE id = 3;
SELECT count(*) FROM orders WHERE customer_id = 7;
SELECT count(*) FROM orders WHERE customer_id = 8;

-- Топ запросов: в реальной жизни сортируют по total_exec_time
SELECT calls, rows, query
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
  AND query ILIKE 'select%orders%' AND query NOT ILIKE '%pg_stat_statements%'
ORDER BY calls DESC, query;

-- Статистика доступа к таблице и индексам (счётчики сбрасываются в общую память не мгновенно)
SELECT pg_stat_force_next_flush();
-- seq_scan учитывает и служебные проходы по таблице (построение индекса и т. п.)
SELECT relname, seq_scan, idx_scan, n_tup_ins FROM pg_stat_user_tables WHERE relname = 'orders';
SELECT indexrelname, idx_scan FROM pg_stat_user_indexes WHERE relname = 'orders' ORDER BY indexrelname;

-- Неиспользуемые индексы — кандидаты на удаление (смотри статистику за долгий период и на всех репликах)
SELECT indexrelid::regclass AS index, pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes s JOIN pg_index i USING (indexrelid)
WHERE s.idx_scan = 0 AND NOT i.indisunique AND s.relname = 'orders';

-- Текущие сессии: состояние и ожидания
SELECT backend_type, state, wait_event_type IS NULL AS not_waiting
FROM pg_stat_activity WHERE pid = pg_backend_pid();

-- Долгие запросы и забытые транзакции (на учебном сервере — пусто)
SELECT pid, state, now() - xact_start AS xact_age, left(query, 60) AS query
FROM pg_stat_activity
WHERE state IN ('active', 'idle in transaction')
  AND now() - xact_start > interval '5 minutes'
ORDER BY xact_start;

-- Размеры: таблица, индексы, TOAST
SELECT pg_size_pretty(pg_table_size('orders')) AS table_size,
       pg_size_pretty(pg_indexes_size('orders')) AS indexes_size,
       pg_size_pretty(pg_total_relation_size('orders')) AS total;
