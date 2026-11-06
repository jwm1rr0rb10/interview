-- Учебный магазин: клиенты, категории (дерево), товары, заказы.
CREATE TABLE customers (id int PRIMARY KEY, name text NOT NULL, city text);
CREATE TABLE categories (id int PRIMARY KEY, parent_id int REFERENCES categories, name text NOT NULL);
CREATE TABLE products (id int PRIMARY KEY, category_id int NOT NULL REFERENCES categories, name text NOT NULL, price numeric(10,2) NOT NULL);
CREATE TABLE orders (id int PRIMARY KEY, customer_id int NOT NULL REFERENCES customers, created_at date NOT NULL);
CREATE TABLE order_items (order_id int REFERENCES orders, product_id int REFERENCES products, qty int NOT NULL, PRIMARY KEY (order_id, product_id));

INSERT INTO customers VALUES (1, 'Аня', 'Москва'), (2, 'Боря', 'Казань'), (3, 'Вика', 'Москва'), (4, 'Гоша', NULL);
INSERT INTO categories VALUES (1, NULL, 'Всё'), (2, 1, 'Посуда'), (3, 1, 'Одежда'), (4, 3, 'Футболки'), (5, 3, 'Головные уборы');
INSERT INTO products VALUES (1, 2, 'Кружка', 450), (2, 2, 'Термос', 1800), (3, 4, 'Футболка', 1290),
                            (4, 5, 'Кепка', 990), (5, 4, 'Лонгслив', 1990);
INSERT INTO orders VALUES (10, 1, '2026-10-01'), (11, 1, '2026-10-15'), (12, 2, '2026-10-20'), (13, 3, '2026-11-02');
INSERT INTO order_items VALUES (10, 1, 2), (10, 3, 1), (11, 2, 1), (12, 1, 1), (12, 4, 2), (13, 5, 1), (13, 1, 3);

-- INNER JOIN: только совпадения; LEFT JOIN: все клиенты, даже без заказов
SELECT c.name, o.id AS order_id, o.created_at
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.id
ORDER BY c.id, o.id;

-- Агрегаты и HAVING: выручка по клиентам, у кого больше одного заказа
SELECT c.name, count(DISTINCT o.id) AS orders, sum(oi.qty * p.price) AS revenue
FROM customers c
JOIN orders o       ON o.customer_id = c.id
JOIN order_items oi ON oi.order_id = o.id
JOIN products p     ON p.id = oi.product_id
GROUP BY c.id, c.name                 -- c.name можно не перечислять: c.id — первичный ключ
HAVING count(DISTINCT o.id) > 1;

-- FILTER: несколько условных агрегатов за один проход
SELECT count(*) AS items,
       count(*) FILTER (WHERE p.price >= 1000) AS expensive,
       sum(oi.qty) FILTER (WHERE p.category_id = 2) AS dishes_qty
FROM order_items oi JOIN products p ON p.id = oi.product_id;

-- Анти-джойн: клиенты без заказов (NOT EXISTS — надёжно с NULL)
SELECT c.name FROM customers c
WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id);

-- Полу-джойн: товары, которые хоть раз покупали
SELECT p.name FROM products p
WHERE EXISTS (SELECT 1 FROM order_items oi WHERE oi.product_id = p.id)
ORDER BY p.id;

-- Скалярный подзапрос в SELECT и сравнение со средним
SELECT name, price, round(price / (SELECT avg(price) FROM products), 2) AS vs_avg
FROM products
WHERE price > (SELECT avg(price) FROM products)
ORDER BY price DESC;

-- LATERAL: для каждого клиента — его последний заказ (top-N на группу)
SELECT c.name, last.id AS last_order, last.created_at
FROM customers c
CROSS JOIN LATERAL (
    SELECT o.id, o.created_at FROM orders o
    WHERE o.customer_id = c.id
    ORDER BY o.created_at DESC
    LIMIT 1
) AS last
ORDER BY c.id;

-- DISTINCT ON: то же самое короче (специфика PostgreSQL)
SELECT DISTINCT ON (o.customer_id) o.customer_id, o.id, o.created_at
FROM orders o
ORDER BY o.customer_id, o.created_at DESC;

-- CTE: именованные шаги запроса
WITH revenue AS (
    SELECT oi.product_id, sum(oi.qty * p.price) AS total
    FROM order_items oi JOIN products p ON p.id = oi.product_id
    GROUP BY oi.product_id
), ranked AS (
    SELECT product_id, total, total * 100 / sum(total) OVER () AS pct
    FROM revenue
)
SELECT p.name, r.total, round(r.pct, 1) AS pct
FROM ranked r JOIN products p ON p.id = r.product_id
ORDER BY r.total DESC;

-- Рекурсивный CTE: дерево категорий с путём и уровнем
WITH RECURSIVE tree AS (
    SELECT id, name, 0 AS depth, name AS path
    FROM categories WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, c.name, t.depth + 1, t.path || ' / ' || c.name
    FROM categories c JOIN tree t ON c.parent_id = t.id
)
SELECT repeat('  ', depth) || name AS category, path FROM tree ORDER BY path;

-- Рекурсия вверх: все товары категории «Одежда» с подкатегориями
WITH RECURSIVE sub AS (
    SELECT id FROM categories WHERE name = 'Одежда'
    UNION ALL
    SELECT c.id FROM categories c JOIN sub ON c.parent_id = sub.id
)
SELECT p.name FROM products p WHERE p.category_id IN (SELECT id FROM sub) ORDER BY p.name;

-- UNION убирает дубликаты (и сортирует/хэширует), UNION ALL — нет
SELECT city FROM customers UNION SELECT 'Москва' ORDER BY 1;
SELECT city FROM customers UNION ALL SELECT 'Москва' ORDER BY 1;

-- Строковые агрегаты и массивы
SELECT o.id, string_agg(p.name, ', ' ORDER BY p.name) AS items, array_agg(oi.qty ORDER BY p.name) AS qtys
FROM orders o JOIN order_items oi ON oi.order_id = o.id JOIN products p ON p.id = oi.product_id
GROUP BY o.id ORDER BY o.id;
