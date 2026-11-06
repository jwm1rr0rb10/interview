-- Роли общие для всего кластера: перед примером убираем следы прошлых запусков
DO $$
DECLARE r text;
BEGIN
    FOREACH r IN ARRAY ARRAY['course_owner', 'course_app', 'course_readonly'] LOOP
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = r) THEN
            EXECUTE format('DROP OWNED BY %I CASCADE', r);
            EXECUTE format('DROP ROLE %I', r);
        END IF;
    END LOOP;
END $$;

-- Владелец схемы, приложение и аналитик только для чтения
CREATE ROLE course_owner NOLOGIN;
CREATE ROLE course_app LOGIN PASSWORD 'change-me';   -- пароль хранится как SCRAM-SHA-256
CREATE ROLE course_readonly NOLOGIN;
SELECT rolname, rolcanlogin, rolsuper FROM pg_roles WHERE rolname LIKE 'course_%' ORDER BY 1;

CREATE SCHEMA shop AUTHORIZATION course_owner;
GRANT USAGE ON SCHEMA shop TO course_app, course_readonly;
-- Права на таблицы, которые владелец создаст в будущем
ALTER DEFAULT PRIVILEGES FOR ROLE course_owner IN SCHEMA shop
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO course_app;
ALTER DEFAULT PRIVILEGES FOR ROLE course_owner IN SCHEMA shop
    GRANT SELECT ON TABLES TO course_readonly;

SET ROLE course_owner;
CREATE TABLE shop.orders (id int PRIMARY KEY, tenant_id int NOT NULL, total int NOT NULL);
INSERT INTO shop.orders VALUES (1, 1, 100), (2, 1, 250), (3, 2, 999);
RESET ROLE;
\dp shop.orders

-- С PostgreSQL 15 обычные роли не могут создавать объекты в схеме public
SET ROLE course_app;
CREATE TABLE public.sneaky (id int);
SELECT count(*) FROM shop.orders;
DELETE FROM shop.orders WHERE id = 999;
RESET ROLE;

SET ROLE course_readonly;
SELECT count(*) FROM shop.orders;
UPDATE shop.orders SET total = 0;
RESET ROLE;

-- Row Level Security: изоляция арендаторов (multi-tenant) на уровне базы
ALTER TABLE shop.orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON shop.orders
    USING (tenant_id = current_setting('app.tenant_id', true)::int)        -- какие строки видны
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::int);  -- какие можно записать

SET ROLE course_app;
SELECT * FROM shop.orders;                         -- арендатор не задан — ничего не видно
SET app.tenant_id = '1';                           -- приложение ставит это в начале запроса (SET LOCAL)
SELECT * FROM shop.orders ORDER BY id;
UPDATE shop.orders SET total = 0 WHERE id = 3;     -- чужая строка невидима: обновлено 0 строк
INSERT INTO shop.orders VALUES (4, 2, 1);          -- запись за другого арендатора запрещена
SET app.tenant_id = '2';
SELECT * FROM shop.orders ORDER BY id;
RESET ROLE;

-- Владелец таблицы обходит RLS, пока не включён FORCE
SET ROLE course_owner;
SELECT count(*) AS owner_sees FROM shop.orders;
RESET ROLE;
ALTER TABLE shop.orders FORCE ROW LEVEL SECURITY;
SET ROLE course_owner;
SELECT count(*) AS owner_sees_with_force FROM shop.orders;
RESET ROLE;

-- Права на отдельные колонки
CREATE TABLE shop.customers (id int PRIMARY KEY, name text, phone text);
GRANT SELECT (id, name) ON shop.customers TO course_readonly;
SET ROLE course_readonly;
SELECT id, name FROM shop.customers;
SELECT phone FROM shop.customers;
RESET ROLE;

-- Встроенные роли вместо суперпользователя для типовых задач
SELECT rolname FROM pg_roles
WHERE rolname IN ('pg_read_all_data', 'pg_write_all_data', 'pg_monitor', 'pg_signal_backend', 'pg_maintain')
ORDER BY 1;

-- Уборка: роль нельзя удалить, пока у неё есть объекты и права
SET client_min_messages = warning;
DROP OWNED BY course_owner CASCADE;
DROP OWNED BY course_app;
DROP OWNED BY course_readonly;
DROP ROLE course_owner, course_app, course_readonly;
