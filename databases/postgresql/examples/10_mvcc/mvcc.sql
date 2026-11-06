-- MVCC своими глазами. Вторую сессию открываем через dblink: это настоящее
-- отдельное подключение к той же базе, как второй терминал с psql.
CREATE EXTENSION dblink;
CREATE EXTENSION pageinspect;
CREATE TABLE accounts (id int PRIMARY KEY, owner text, balance int NOT NULL);
INSERT INTO accounts VALUES (1, 'Аня', 100), (2, 'Боря', 100);

-- Скрытые колонки: ctid — физический адрес версии (страница, позиция), xmax = 0 — версия жива
SELECT ctid, xmax, id, balance FROM accounts ORDER BY id;

-- UPDATE не меняет строку на месте, а создаёт новую версию
UPDATE accounts SET balance = 90 WHERE id = 1;
SELECT ctid, id, balance FROM accounts ORDER BY id;

-- На странице теперь три версии: старая версия Ани помечена удалившей её транзакцией
SELECT lp, t_ctid, t_xmax <> 0 AS has_xmax
FROM heap_page_items(get_raw_page('accounts', 0)) ORDER BY lp;

-- Вторая сессия «b»
SELECT dblink_connect('b', format('dbname=%s user=%s', current_database(), current_user));

-- 1. Незакоммиченные изменения не видны другим: грязного чтения в PostgreSQL нет
BEGIN;
UPDATE accounts SET balance = balance - 30 WHERE id = 1;
SELECT balance AS a_sees FROM accounts WHERE id = 1;
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_sees int);
COMMIT;
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_sees_after_commit int);

-- 2. READ COMMITTED (по умолчанию): каждый запрос видит свежий снимок — неповторяющееся чтение
SELECT dblink_exec('b', 'BEGIN ISOLATION LEVEL READ COMMITTED');
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_first_read int);
UPDATE accounts SET balance = 50 WHERE id = 1;          -- сессия A меняет и коммитит
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_second_read int);
SELECT dblink_exec('b', 'COMMIT');

-- 3. REPEATABLE READ: снимок берётся один раз на всю транзакцию
SELECT dblink_exec('b', 'BEGIN ISOLATION LEVEL REPEATABLE READ');
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_first_read int);
UPDATE accounts SET balance = 40 WHERE id = 1;
SELECT * FROM dblink('b', 'SELECT balance FROM accounts WHERE id = 1') AS t(b_second_read int);
-- ...но изменить строку, которую после снимка изменил кто-то другой, нельзя:
SELECT dblink_exec('b', 'UPDATE accounts SET balance = balance + 1 WHERE id = 1');
SELECT dblink_exec('b', 'ROLLBACK');

-- 4. Аномалия write skew: правило «хотя бы один врач на дежурстве»
CREATE TABLE oncall (doctor text PRIMARY KEY, on_duty boolean NOT NULL);
INSERT INTO oncall VALUES ('Алиса', true), ('Борис', true);
SELECT dblink_connect('c', format('dbname=%s user=%s', current_database(), current_user));

-- REPEATABLE READ: каждый видит двух дежурных, каждый уходит — нарушение правила
SELECT dblink_exec('b', 'BEGIN ISOLATION LEVEL REPEATABLE READ');
SELECT dblink_exec('c', 'BEGIN ISOLATION LEVEL REPEATABLE READ');
SELECT * FROM dblink('b', 'SELECT count(*) FROM oncall WHERE on_duty') AS t(b_sees int);
SELECT * FROM dblink('c', 'SELECT count(*) FROM oncall WHERE on_duty') AS t(c_sees int);
SELECT dblink_exec('b', $$UPDATE oncall SET on_duty = false WHERE doctor = 'Алиса'$$);
SELECT dblink_exec('c', $$UPDATE oncall SET on_duty = false WHERE doctor = 'Борис'$$);
SELECT dblink_exec('b', 'COMMIT');
SELECT dblink_exec('c', 'COMMIT');
SELECT count(*) FILTER (WHERE on_duty) AS on_duty_now FROM oncall;

-- SERIALIZABLE: тот же сценарий, но вторая транзакция получает ошибку сериализации
UPDATE oncall SET on_duty = true;
SELECT dblink_exec('b', 'BEGIN ISOLATION LEVEL SERIALIZABLE');
SELECT dblink_exec('c', 'BEGIN ISOLATION LEVEL SERIALIZABLE');
SELECT * FROM dblink('b', 'SELECT count(*) FROM oncall WHERE on_duty') AS t(b_sees int);
SELECT * FROM dblink('c', 'SELECT count(*) FROM oncall WHERE on_duty') AS t(c_sees int);
SELECT dblink_exec('b', $$UPDATE oncall SET on_duty = false WHERE doctor = 'Алиса'$$);
SELECT dblink_exec('c', $$UPDATE oncall SET on_duty = false WHERE doctor = 'Борис'$$);
SELECT dblink_exec('b', 'COMMIT');
SELECT dblink_exec('c', 'COMMIT');
SELECT dblink_exec('c', 'ROLLBACK');
SELECT count(*) FILTER (WHERE on_duty) AS on_duty_now FROM oncall;

SELECT dblink_disconnect('b'), dblink_disconnect('c');
