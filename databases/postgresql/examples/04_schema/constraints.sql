-- Ограничения: база сама охраняет инварианты данных.
CREATE TABLE customers (
    id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL UNIQUE,
    name  text NOT NULL CHECK (name <> '')
);

CREATE TABLE orders (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id bigint NOT NULL REFERENCES customers (id) ON DELETE RESTRICT,
    status      text NOT NULL DEFAULT 'new'
                CHECK (status IN ('new', 'paid', 'shipped', 'cancelled')),
    total       numeric(12, 2) NOT NULL CHECK (total >= 0),
    paid_at     timestamptz,
    CONSTRAINT paid_has_time CHECK (status <> 'paid' OR paid_at IS NOT NULL)
);
-- Внешний ключ не создаёт индекс автоматически — создаём сами
CREATE INDEX orders_customer_id_idx ON orders (customer_id);

CREATE TABLE order_items (
    order_id bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    sku      text   NOT NULL,
    qty      int    NOT NULL CHECK (qty > 0),
    PRIMARY KEY (order_id, sku)                -- составной ключ: позиция уникальна в заказе
);

INSERT INTO customers (email, name) VALUES ('anya@example.com', 'Аня'), ('borya@example.com', 'Боря');
INSERT INTO orders (customer_id, total) VALUES (1, 1500), (1, 300);
INSERT INTO order_items VALUES (1, 'mug', 2), (1, 'tee', 1);

-- Каждое нарушение — понятная ошибка с именем ограничения
INSERT INTO customers (email, name) VALUES ('anya@example.com', 'Другая Аня');
INSERT INTO orders (customer_id, total) VALUES (42, 100);
INSERT INTO orders (customer_id, total) VALUES (2, -5);
UPDATE orders SET status = 'paid' WHERE id = 1;
INSERT INTO order_items VALUES (1, 'mug', 1);
DELETE FROM customers WHERE id = 1;

-- ON DELETE CASCADE: удаление заказа удаляет его позиции
DELETE FROM orders WHERE id = 1;
SELECT count(*) AS items_left FROM order_items;

-- UNIQUE и NULL: по умолчанию NULL не равны друг другу
CREATE TABLE coupons (code text, campaign text, UNIQUE (code));
INSERT INTO coupons VALUES (NULL, 'a'), (NULL, 'b');      -- оба проходят
CREATE TABLE coupons2 (code text, UNIQUE NULLS NOT DISTINCT (code));
INSERT INTO coupons2 VALUES (NULL), (NULL);                -- PostgreSQL 15+: второй NULL — дубликат

-- Частичная уникальность: один активный email, удалённые могут повторяться
CREATE TABLE accounts (email text NOT NULL, deleted_at timestamptz);
CREATE UNIQUE INDEX accounts_email_active_uq ON accounts (email) WHERE deleted_at IS NULL;
INSERT INTO accounts VALUES ('a@x.ru', '2026-01-01'), ('a@x.ru', NULL);
INSERT INTO accounts VALUES ('a@x.ru', NULL);

-- EXCLUDE: две брони одной переговорки не могут пересекаться по времени
CREATE EXTENSION btree_gist;   -- позволяет использовать = для обычных типов в GiST
CREATE TABLE room_bookings (
    room   int NOT NULL,
    during tstzrange NOT NULL,
    EXCLUDE USING gist (room WITH =, during WITH &&)
);
INSERT INTO room_bookings VALUES (1, '[2026-11-06 10:00+00, 2026-11-06 11:00+00)'),
                                 (1, '[2026-11-06 11:00+00, 2026-11-06 12:00+00)'),  -- стык — можно
                                 (2, '[2026-11-06 10:30+00, 2026-11-06 11:30+00)');
INSERT INTO room_bookings VALUES (1, '[2026-11-06 10:30+00, 2026-11-06 11:30+00)');

-- Темпоральные ключи (PostgreSQL 18): уникальность «во времени» и внешние ключи с PERIOD
CREATE TABLE prices (
    sku      text NOT NULL,
    valid    daterange NOT NULL,
    price    numeric(10, 2) NOT NULL,
    PRIMARY KEY (sku, valid WITHOUT OVERLAPS)
);
INSERT INTO prices VALUES ('mug', '[2026-01-01, 2026-07-01)', 450), ('mug', '[2026-07-01,)', 495);
INSERT INTO prices VALUES ('mug', '[2026-06-01, 2026-08-01)', 470);

CREATE TABLE promo (
    sku    text NOT NULL,
    period daterange NOT NULL,
    FOREIGN KEY (sku, PERIOD period) REFERENCES prices (sku, PERIOD valid)
);
INSERT INTO promo VALUES ('mug', '[2026-06-15, 2026-07-15)');   -- покрыто двумя периодами цен — ок
INSERT INTO promo VALUES ('mug', '[2025-12-01, 2026-01-15)');   -- декабря 2025 в ценах нет

-- Отложенная проверка: ограничение проверяется в момент COMMIT
CREATE TABLE seats (id int PRIMARY KEY, pos int NOT NULL,
                    CONSTRAINT seats_pos_uq UNIQUE (pos) DEFERRABLE INITIALLY IMMEDIATE);
INSERT INTO seats VALUES (1, 1), (2, 2);
BEGIN;
SET CONSTRAINTS seats_pos_uq DEFERRED;
UPDATE seats SET pos = 2 WHERE id = 1;   -- временно дубликат — проверка отложена
UPDATE seats SET pos = 1 WHERE id = 2;
COMMIT;
SELECT * FROM seats ORDER BY id;

-- NOT VALID: новое ограничение на большой таблице без долгой проверки старых строк
ALTER TABLE orders ADD CONSTRAINT total_limit CHECK (total < 1000000) NOT VALID;
ALTER TABLE orders VALIDATE CONSTRAINT total_limit;   -- проверка без блокировки записи
\d orders
