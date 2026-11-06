SET max_parallel_workers_per_gather = 0;
-- Таблица событий, разбитая по месяцам
CREATE TABLE events (
    id         bigint GENERATED ALWAYS AS IDENTITY,
    created_at timestamptz NOT NULL,
    kind       text NOT NULL,
    PRIMARY KEY (id, created_at)          -- ключ партиционирования обязан входить в PK и UNIQUE
) PARTITION BY RANGE (created_at);

CREATE TABLE events_2026_09 PARTITION OF events FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE events_2026_10 PARTITION OF events FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE events_2026_11 PARTITION OF events FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE events_default PARTITION OF events DEFAULT;

-- Индекс на родителе автоматически создаётся на каждой секции
CREATE INDEX events_kind_created_idx ON events (kind, created_at);

INSERT INTO events (created_at, kind)
SELECT '2026-09-01'::timestamptz + (i || ' minutes')::interval,
       (ARRAY['click', 'view', 'buy'])[i % 3 + 1]
FROM generate_series(0, 125000) i;
ANALYZE events;

SELECT tableoid::regclass AS partition, count(*), min(created_at)::date, max(created_at)::date
FROM events GROUP BY 1 ORDER BY 1;

-- Отсечение секций (partition pruning): читается только октябрь
EXPLAIN (COSTS OFF)
SELECT count(*) FROM events WHERE created_at >= '2026-10-10' AND created_at < '2026-10-11';

-- Без условия на ключ партиционирования читаются все секции
EXPLAIN (COSTS OFF) SELECT count(*) FROM events WHERE kind = 'buy';

-- Отсечение во время выполнения: значение параметра неизвестно при планировании
PREPARE by_day(timestamptz) AS SELECT count(*) FROM events WHERE created_at >= $1 AND created_at < $1 + interval '1 day';
SET plan_cache_mode = force_generic_plan;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF) EXECUTE by_day('2026-11-05');
RESET plan_cache_mode;

-- Уникальность без ключа партиционирования невозможна
ALTER TABLE events ADD CONSTRAINT events_id_uq UNIQUE (id);

-- Строка вне всех диапазонов попала в DEFAULT; теперь секция на этот месяц не создаётся
INSERT INTO events (created_at, kind) VALUES ('2026-12-15', 'view');
CREATE TABLE events_2026_12 PARTITION OF events FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Хранение данных за N месяцев: удаление месяца — отсоединить и удалить секцию, а не DELETE
ALTER TABLE events DETACH PARTITION events_2026_09;   -- на проде: DETACH ... CONCURRENTLY
DROP TABLE events_2026_09;
SELECT count(*) AS left_rows FROM events;

-- Список партиций
SELECT relid::text AS partition, isleaf, level FROM pg_partition_tree('events') ORDER BY 1;

-- Секционирование по списку и по хешу
CREATE TABLE customers (id bigint, region text NOT NULL) PARTITION BY LIST (region);
CREATE TABLE customers_ru PARTITION OF customers FOR VALUES IN ('ru', 'by', 'kz');
CREATE TABLE customers_eu PARTITION OF customers FOR VALUES IN ('de', 'fr');
CREATE TABLE sessions (id bigint, user_id bigint) PARTITION BY HASH (user_id);
CREATE TABLE sessions_p0 PARTITION OF sessions FOR VALUES WITH (MODULUS 2, REMAINDER 0);
CREATE TABLE sessions_p1 PARTITION OF sessions FOR VALUES WITH (MODULUS 2, REMAINDER 1);
INSERT INTO sessions SELECT i, i FROM generate_series(1, 1000) i;
SELECT tableoid::regclass, count(*) FROM sessions GROUP BY 1 ORDER BY 1;
