-- CRUD, RETURNING, upsert и MERGE.
CREATE TABLE products (
    sku        text PRIMARY KEY,
    name       text NOT NULL,
    price      numeric(10, 2) NOT NULL CHECK (price >= 0),
    stock      int NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT '2026-11-01 10:00:00+00'
);

-- INSERT ... RETURNING возвращает вставленные строки: не нужен второй SELECT
INSERT INTO products (sku, name, price, stock) VALUES
    ('mug-01', 'Кружка', 450, 10),
    ('tee-02', 'Футболка', 1290, 0),
    ('cap-03', 'Кепка', 990, 5)
RETURNING sku, price;

-- UPDATE ... RETURNING со старым и новым значением (OLD/NEW — PostgreSQL 18)
UPDATE products
   SET price = price * 1.10
 WHERE sku = 'mug-01'
RETURNING sku, old.price AS old_price, new.price AS new_price;

-- Повторная вставка того же ключа — ошибка уникальности
INSERT INTO products (sku, name, price) VALUES ('mug-01', 'Кружка', 500);

-- Upsert: вставить или обновить, атомарно и без гонок
INSERT INTO products (sku, name, price, stock)
VALUES ('mug-01', 'Кружка', 500, 3), ('bag-04', 'Сумка', 1990, 2)
ON CONFLICT (sku) DO UPDATE
   SET stock = products.stock + excluded.stock,   -- excluded — строка, которую пытались вставить
       price = excluded.price
RETURNING sku, stock, price, (xmax <> 0) AS was_updated;

-- MERGE: синхронизация с «поставкой» одной командой; RETURNING у MERGE — с PostgreSQL 17
CREATE TABLE delivery (sku text, qty int);
INSERT INTO delivery VALUES ('tee-02', 20), ('cap-03', -5), ('hat-05', 7);

MERGE INTO products p
USING delivery d ON p.sku = d.sku
WHEN MATCHED AND p.stock + d.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET stock = p.stock + d.qty
WHEN NOT MATCHED THEN INSERT (sku, name, price, stock) VALUES (d.sku, 'Новинка', 0, d.qty)
RETURNING merge_action(), p.sku, p.stock;

-- DELETE ... RETURNING: удалить и сразу получить удалённое
DELETE FROM products WHERE price = 0 RETURNING sku;

SELECT sku, name, price, stock FROM products ORDER BY sku;
