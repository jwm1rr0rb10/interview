-- Числа: точные и приближённые
SELECT 0.1::float8 + 0.2::float8 AS float_sum,
       0.1::numeric + 0.2::numeric AS numeric_sum,
       (0.1::float8 + 0.2::float8) = 0.3::float8 AS float_equal;

SELECT 2147483647::int + 1;            -- переполнение int — ошибка, а не «тихий» перенос
SELECT 2147483647::bigint + 1 AS ok;
SELECT 7 / 2 AS int_div, 7 / 2.0 AS numeric_div, round(10 / 3.0, 2) AS rounded;

-- Строки: text, varchar(n) и проверка длины
CREATE TABLE t_strings (code varchar(3), note text CHECK (length(note) <= 10));
INSERT INTO t_strings VALUES ('ABCD', 'ok');
INSERT INTO t_strings VALUES ('ABC', 'слишком длинный текст');
SELECT length('Привет') AS chars, octet_length('Привет') AS bytes;

-- Время: timestamptz хранит момент времени, timestamp — «показания часов» без пояса
SELECT '2026-11-06 12:00:00+03'::timestamptz AS tz_utc_display,
       '2026-11-06 12:00:00+03'::timestamp   AS no_tz_ignored_offset;

SET timezone = 'Europe/Moscow';
SELECT '2026-11-06 09:00:00+00'::timestamptz AS shown_in_moscow;
SELECT '2026-11-06 09:00:00+00'::timestamptz AT TIME ZONE 'Asia/Vladivostok' AS wall_clock_vladivostok;
RESET timezone;

-- Интервалы и календарная арифметика
SELECT '2026-01-31'::date + interval '1 month' AS plus_month,
       '2026-03-01'::date - '2026-02-01'::date AS days_in_feb,
       age('2026-11-06'::date, '1990-05-15'::date) AS age,
       date_trunc('month', '2026-11-06 15:30:00+00'::timestamptz) AS month_start;

SELECT d::date AS day, to_char(d, 'Dy') AS dow
FROM generate_series('2026-11-01'::date, '2026-11-05'::date, interval '1 day') AS d;

-- UUID: uuidv7() (PostgreSQL 18) начинается с метки времени — хорошо для первичных ключей
SELECT uuid_extract_version(uuidv7()) AS v7, uuid_extract_version(gen_random_uuid()) AS v4,
       uuidv7() < uuidv7() AS v7_sortable,
       uuid_extract_timestamp(uuidv7()) > '2026-01-01' AS has_timestamp;

-- Identity против serial
CREATE TABLE t_identity (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, v text);
INSERT INTO t_identity (v) VALUES ('a') RETURNING id;
INSERT INTO t_identity (id, v) VALUES (100, 'b');          -- ALWAYS не даёт задать id вручную
INSERT INTO t_identity (id, v) OVERRIDING SYSTEM VALUE VALUES (100, 'b') RETURNING id;

-- Генерируемые колонки: VIRTUAL (по умолчанию с PostgreSQL 18) и STORED
CREATE TABLE t_orders (
    qty        int NOT NULL,
    unit_price numeric(10, 2) NOT NULL,
    total      numeric(12, 2) GENERATED ALWAYS AS (qty * unit_price),          -- VIRTUAL: вычисляется при чтении
    total_vat  numeric(12, 2) GENERATED ALWAYS AS (qty * unit_price * 1.2) STORED  -- хранится на диске
);
INSERT INTO t_orders (qty, unit_price) VALUES (3, 199.90);
SELECT * FROM t_orders;
\d t_orders

-- Массивы
SELECT ARRAY['go', 'sql', 'pg'] AS tags,
       'sql' = ANY (ARRAY['go', 'sql']) AS has_sql,
       ARRAY[1, 2, 3] @> ARRAY[2] AS contains,
       array_length(ARRAY[1, 2, 3], 1) AS len,
       (ARRAY[10, 20, 30])[1] AS first_is_index_1;
SELECT unnest(ARRAY['a', 'b']) AS letter;

-- Перечисления и домены
CREATE TYPE order_status AS ENUM ('new', 'paid', 'shipped', 'cancelled');
CREATE DOMAIN email AS text CHECK (VALUE ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$');
CREATE TABLE t_enum (status order_status, contact email);
INSERT INTO t_enum VALUES ('paid', 'anya@example.com');
INSERT INTO t_enum VALUES ('lost', 'anya@example.com');
INSERT INTO t_enum VALUES ('new', 'not-an-email');
SELECT 'paid'::order_status < 'shipped'::order_status AS enum_ordered;

-- Диапазоны и мультидиапазоны
SELECT int4range(10, 20) AS r, int4range(10, 20) @> 15 AS has_15, int4range(10, 20) @> 20 AS has_20,
       tstzrange('2026-11-06 10:00+00', '2026-11-06 12:00+00') && tstzrange('2026-11-06 11:00+00', '2026-11-06 13:00+00') AS overlaps,
       int4multirange(int4range(1, 5), int4range(3, 8), int4range(10, 12)) AS merged;

-- Булевы значения
SELECT 'yes'::boolean, 'off'::boolean, 't'::boolean AND NOT false AS expr;
