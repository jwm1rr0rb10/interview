CREATE EXTENSION pgstattuple;
CREATE EXTENSION pageinspect;
CREATE EXTENSION pg_visibility;
-- autovacuum выключен для таблицы, чтобы наблюдать мёртвые версии без помех
CREATE TABLE items (id int PRIMARY KEY, qty int NOT NULL, note text NOT NULL)
WITH (autovacuum_enabled = off);
INSERT INTO items SELECT i, 0, repeat('x', 50) FROM generate_series(1, 100000) i;

CREATE VIEW items_stats AS
SELECT pg_size_pretty(pg_relation_size('items')) AS size, tuple_count AS live,
       dead_tuple_count AS dead, round(free_percent) AS free_pct
FROM pgstattuple('items');

SELECT * FROM items_stats;

-- Обновили все строки: каждая получила новую версию, старые стали мёртвыми
UPDATE items SET qty = qty + 1;
SELECT * FROM items_stats;

-- VACUUM освобождает место под новые версии, но файл таблицы не уменьшает
VACUUM items;
SELECT * FROM items_stats;

-- Освободившееся место переиспользуется: ещё одно полное обновление не увеличивает файл
UPDATE items SET qty = qty + 1;
VACUUM items;
SELECT * FROM items_stats;

-- VACUUM FULL переписывает таблицу целиком и возвращает место ОС — под эксклюзивной блокировкой
VACUUM FULL items;
SELECT * FROM items_stats;

-- HOT-обновления: если не меняются индексированные колонки и на странице есть место,
-- новая версия ложится на ту же страницу, а индексы не трогаются
CREATE TABLE hot_demo (id int PRIMARY KEY, counter int NOT NULL, label text NOT NULL)
WITH (fillfactor = 70, autovacuum_enabled = off);   -- 30% страницы оставлено под обновления
CREATE INDEX hot_demo_label_idx ON hot_demo (label);
INSERT INTO hot_demo SELECT i, 0, 'L' || i FROM generate_series(1, 50) i;

UPDATE hot_demo SET counter = counter + 1 WHERE id <= 10;      -- неиндексированная колонка
UPDATE hot_demo SET label = label || '!' WHERE id BETWEEN 11 AND 20;  -- индексированная колонка
SELECT count(*) FILTER (WHERE t_infomask2 & 32768 <> 0) AS heap_only_tuples   -- флаг HEAP_ONLY_TUPLE
FROM heap_page_items(get_raw_page('hot_demo', 0));

-- Карта видимости и заморозка. VACUUM FULL выше переписал таблицу и не заполнил карту видимости
SELECT count(*) FILTER (WHERE all_visible) AS all_visible, count(*) FILTER (WHERE all_frozen) AS all_frozen
FROM pg_visibility_map('items');
VACUUM (FREEZE) items;   -- обычный VACUUM отметит страницы видимыми, FREEZE — ещё и замороженными
SELECT count(*) FILTER (WHERE all_visible) AS all_visible, count(*) FILTER (WHERE all_frozen) AS all_frozen
FROM pg_visibility_map('items');

-- Возраст самой старой незамороженной транзакции: до 2 млрд — авария (wraparound)
SELECT age(datfrozenxid) < 200000000 AS far_from_forced_freeze FROM pg_database WHERE datname = current_database();

-- Настройки autovacuum по умолчанию (PostgreSQL 18)
SELECT name, setting, unit FROM pg_settings
WHERE name IN ('autovacuum_vacuum_threshold', 'autovacuum_vacuum_scale_factor', 'autovacuum_vacuum_max_threshold',
               'autovacuum_vacuum_insert_scale_factor', 'autovacuum_analyze_scale_factor', 'autovacuum_naptime',
               'autovacuum_max_workers', 'autovacuum_vacuum_cost_limit', 'autovacuum_freeze_max_age')
ORDER BY name;

-- Для большой и активной таблицы порог задают отдельно
ALTER TABLE items SET (autovacuum_enabled = on, autovacuum_vacuum_scale_factor = 0.01, autovacuum_vacuum_threshold = 1000);
SELECT reloptions FROM pg_class WHERE relname = 'items';
