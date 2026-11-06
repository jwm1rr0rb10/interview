-- jsonb: бинарный JSON с индексами и операторами
SELECT '{"b": 1, "a": 2, "a": 3}'::json  AS json_keeps_text,
       '{"b": 1, "a": 2, "a": 3}'::jsonb AS jsonb_normalized;

CREATE TABLE events (
    id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind    text NOT NULL,
    payload jsonb NOT NULL
);
INSERT INTO events (kind, payload) VALUES
 ('order', '{"order_id": 1, "total": 1500, "items": [{"sku": "mug", "qty": 2}, {"sku": "tee", "qty": 1}], "customer": {"city": "Москва", "vip": true}}'),
 ('order', '{"order_id": 2, "total": 300,  "items": [{"sku": "mug", "qty": 1}], "customer": {"city": "Казань"}}'),
 ('login', '{"user": "anya", "device": {"os": "ios", "version": "18.1"}}');

-- Доступ к полям: -> возвращает jsonb, ->> — текст, #>> — путь
SELECT id, payload -> 'customer' AS customer, payload ->> 'total' AS total_text,
       (payload ->> 'total')::numeric * 1.2 AS total_vat, payload #>> '{customer,city}' AS city
FROM events WHERE kind = 'order' ORDER BY id;

-- Синтаксис индексов (PostgreSQL 14+)
SELECT payload['customer']['city'] AS city FROM events WHERE id = 1;

-- Содержит (@>) и существование ключа (?)
SELECT id FROM events WHERE payload @> '{"customer": {"vip": true}}';
SELECT id FROM events WHERE payload ? 'device';

-- Изменение: jsonb_set, конкатенация ||, удаление ключа -
UPDATE events SET payload = jsonb_set(payload, '{customer,vip}', 'false') WHERE id = 2;
UPDATE events SET payload = (payload || '{"source": "app"}') - 'device' WHERE id = 3;
SELECT id, payload FROM events WHERE id IN (2, 3) ORDER BY id;

-- Развернуть массив в строки
SELECT e.id, item ->> 'sku' AS sku, (item ->> 'qty')::int AS qty
FROM events e, jsonb_array_elements(e.payload -> 'items') AS item
ORDER BY e.id, sku;

-- SQL/JSON path: фильтры внутри документа
SELECT id, jsonb_path_query_array(payload, '$.items[*] ? (@.qty >= 2).sku') AS bulk_skus
FROM events WHERE kind = 'order' ORDER BY id;
SELECT id FROM events WHERE payload @? '$.items[*] ? (@.sku == "tee")';

-- Стандартные функции SQL/JSON (PostgreSQL 17)
SELECT JSON_VALUE(payload, '$.customer.city') AS city,
       JSON_QUERY(payload, '$.items[0]') AS first_item,
       JSON_EXISTS(payload, '$.customer.vip') AS has_vip
FROM events WHERE id = 1;

-- JSON_TABLE (PostgreSQL 17): JSON в обычную таблицу
SELECT jt.*
FROM events e,
     JSON_TABLE(e.payload, '$' COLUMNS (
         order_id int PATH '$.order_id',
         city text PATH '$.customer.city',
         NESTED PATH '$.items[*]' COLUMNS (sku text PATH '$.sku', qty int PATH '$.qty')
     )) AS jt
WHERE e.kind = 'order'
ORDER BY order_id, sku;

-- Сборка JSON из строк — ответ API прямо из базы
SELECT jsonb_build_object('kind', kind, 'count', count(*), 'ids', jsonb_agg(id ORDER BY id)) AS summary
FROM events GROUP BY kind ORDER BY kind;

-- Индексы: GIN для @> и ?, B-tree по выражению для одного поля
INSERT INTO events (kind, payload)
SELECT 'order', jsonb_build_object('order_id', i, 'total', i % 5000,
       'customer', jsonb_build_object('city', (ARRAY['Москва','Казань','Пермь','Сочи'])[i % 4 + 1], 'vip', i % 100 = 0))
FROM generate_series(3, 50000) i;
CREATE INDEX events_payload_gin ON events USING gin (payload jsonb_path_ops);
CREATE INDEX events_total_idx ON events (((payload ->> 'total')::int));
ANALYZE events;

EXPLAIN (COSTS OFF) SELECT count(*) FROM events WHERE payload @> '{"customer": {"vip": true}}';
EXPLAIN (COSTS OFF) SELECT id FROM events WHERE (payload ->> 'total')::int = 4999;
-- Тот же смысл, но выражение не совпадает с индексом — индекс не используется
EXPLAIN (COSTS OFF) SELECT id FROM events WHERE payload ->> 'total' = '4999';
