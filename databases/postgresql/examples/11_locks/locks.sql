CREATE EXTENSION dblink;
CREATE TABLE accounts (id int PRIMARY KEY, balance int NOT NULL);
INSERT INTO accounts VALUES (1, 100), (2, 100), (3, 100);
SELECT dblink_connect('b', format('dbname=%s user=%s application_name=session_b', current_database(), current_user));

-- 1. Блокировка строки: UPDATE держит её до конца транзакции
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 1;
SELECT dblink_exec('b', $$SET lock_timeout = '200ms'$$);
-- fail_on_error = false: ошибка сессии B придёт как NOTICE и не прервёт нашу транзакцию A
SELECT dblink_exec('b', 'UPDATE accounts SET balance = 0 WHERE id = 1', false);   -- ждёт 200 мс и сдаётся
SELECT dblink_exec('b', 'UPDATE accounts SET balance = balance + 1 WHERE id = 2'); -- другая строка свободна
SELECT * FROM dblink('b', 'SELECT id FROM accounts WHERE id = 1 FOR UPDATE NOWAIT', false) AS t(id int);

-- Какие блокировки держит наша транзакция
SELECT relation::regclass AS relation, mode, granted
FROM pg_locks
WHERE pid = pg_backend_pid() AND locktype = 'relation' AND relation::regclass::text LIKE 'accounts%'
ORDER BY 1, 2;

-- Кто кого ждёт: отправляем запрос сессии B асинхронно, он повиснет на блокировке
SELECT dblink_exec('b', 'RESET lock_timeout');
SELECT dblink_send_query('b', 'UPDATE accounts SET balance = balance + 5 WHERE id = 1');
SELECT pg_sleep(1);   -- дать сессии B встать в ожидание
SELECT application_name, state, wait_event_type, wait_event,
       pg_blocking_pids(pid) = ARRAY[pg_backend_pid()] AS blocked_by_us
FROM pg_stat_activity WHERE application_name = 'session_b';
COMMIT;   -- отпускаем блокировку — запрос B выполняется
SELECT * FROM dblink_get_result('b') AS t(status text);
SELECT * FROM dblink_get_result('b') AS t(status text);
SELECT id, balance FROM accounts ORDER BY id;

-- 2. Очередь задач на FOR UPDATE SKIP LOCKED: воркеры не мешают друг другу
CREATE TABLE jobs (id int PRIMARY KEY, status text NOT NULL DEFAULT 'pending');
INSERT INTO jobs SELECT generate_series(1, 5);
BEGIN;                                        -- воркер A
SELECT id FROM jobs WHERE status = 'pending' ORDER BY id LIMIT 2 FOR UPDATE SKIP LOCKED;
SELECT dblink_exec('b', 'BEGIN');             -- воркер B берёт следующие, не дожидаясь A
SELECT * FROM dblink('b', $$SELECT id FROM jobs WHERE status = 'pending' ORDER BY id LIMIT 2 FOR UPDATE SKIP LOCKED$$) AS t(id int);
UPDATE jobs SET status = 'done' WHERE id IN (1, 2);
COMMIT;
SELECT dblink_exec('b', $$UPDATE jobs SET status = 'done' WHERE id IN (3, 4)$$);
SELECT dblink_exec('b', 'COMMIT');
SELECT id, status FROM jobs ORDER BY id;

-- 3. Дедлок: A ждёт B, B ждёт A — PostgreSQL обнаруживает цикл и прерывает одну транзакцию
BEGIN;
UPDATE accounts SET balance = balance - 1 WHERE id = 1;                       -- A держит строку 1
SELECT dblink_exec('b', 'BEGIN');
SELECT dblink_exec('b', 'UPDATE accounts SET balance = balance - 1 WHERE id = 2');  -- B держит строку 2
SELECT dblink_send_query('b', 'UPDATE accounts SET balance = balance + 1 WHERE id = 1');  -- B ждёт A
SET deadlock_timeout = '100ms';   -- проверка на дедлок через 100 мс ожидания (по умолчанию 1 с)
\set VERBOSITY terse
UPDATE accounts SET balance = balance + 1 WHERE id = 2;                       -- A ждёт B: цикл
\set VERBOSITY default
ROLLBACK;
SELECT * FROM dblink_get_result('b') AS t(status text);                       -- B продолжил работу
SELECT * FROM dblink_get_result('b') AS t(status text);
SELECT dblink_exec('b', 'COMMIT');

-- 4. DDL и очередь блокировок: ALTER TABLE ждёт читателя и блокирует всех после себя
BEGIN;
SELECT count(*) FROM accounts;            -- A держит AccessShareLock до конца транзакции
SELECT dblink_exec('b', $$SET lock_timeout = '200ms'$$);
SELECT dblink_exec('b', 'ALTER TABLE accounts ADD COLUMN note text', false);   -- без lock_timeout висел бы и держал очередь
COMMIT;
SELECT dblink_exec('b', 'ALTER TABLE accounts ADD COLUMN note text');   -- после COMMIT проходит мгновенно

-- 5. Advisory-блокировки: мьютекс по произвольному числу, например «одна задача импорта за раз»
SELECT pg_try_advisory_lock(42) AS a_got_lock;
SELECT * FROM dblink('b', 'SELECT pg_try_advisory_lock(42)') AS t(b_got_lock boolean);
SELECT pg_advisory_unlock(42) AS a_released;
SELECT * FROM dblink('b', 'SELECT pg_try_advisory_lock(42)') AS t(b_got_lock boolean);
SELECT dblink_disconnect('b');
