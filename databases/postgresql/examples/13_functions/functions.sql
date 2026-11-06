-- SQL-функция: простое выражение, планировщик может встроить её в запрос
CREATE FUNCTION vat(amount numeric, rate numeric DEFAULT 0.20)
RETURNS numeric LANGUAGE sql IMMUTABLE
RETURN round(amount * rate, 2);
SELECT vat(1000) AS default_rate, vat(1000, 0.10) AS reduced;

-- PL/pgSQL: переменные, условия, исключения
CREATE TABLE accounts (id int PRIMARY KEY, owner text NOT NULL, balance numeric(12,2) NOT NULL CHECK (balance >= 0));
INSERT INTO accounts VALUES (1, 'Аня', 1000), (2, 'Боря', 100);

CREATE FUNCTION transfer(from_id int, to_id int, amount numeric)
RETURNS numeric LANGUAGE plpgsql AS $$
DECLARE
    new_balance numeric;
BEGIN
    IF amount <= 0 THEN
        RAISE EXCEPTION 'сумма должна быть положительной: %', amount
              USING ERRCODE = 'check_violation', HINT = 'передайте amount > 0';
    END IF;
    -- блокируем строки в одинаковом порядке, чтобы не было дедлоков
    PERFORM 1 FROM accounts WHERE id IN (from_id, to_id) ORDER BY id FOR UPDATE;

    UPDATE accounts SET balance = balance - amount WHERE id = from_id
    RETURNING balance INTO new_balance;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'счёт % не найден', from_id USING ERRCODE = 'no_data_found';
    END IF;
    UPDATE accounts SET balance = balance + amount WHERE id = to_id;
    RETURN new_balance;
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'перевод отклонён: %', SQLERRM;
        RETURN NULL;
END $$;

SELECT transfer(1, 2, 300) AS anya_left;
SELECT transfer(2, 1, 10000) AS rejected;     -- CHECK (balance >= 0) нарушен — перехвачено
SELECT transfer(1, 2, -5) AS negative;
SELECT transfer(99, 1, 5);
SELECT id, balance FROM accounts ORDER BY id;

-- Функция, возвращающая таблицу
CREATE FUNCTION rich_accounts(min_balance numeric)
RETURNS TABLE (owner text, balance numeric) LANGUAGE sql STABLE
AS $$ SELECT owner, balance FROM accounts WHERE balance >= min_balance ORDER BY balance DESC $$;
SELECT * FROM rich_accounts(500);

-- Процедура может фиксировать транзакции внутри — пакетная обработка большими порциями
CREATE TABLE big_log (id int PRIMARY KEY, archived boolean NOT NULL DEFAULT false);
INSERT INTO big_log SELECT generate_series(1, 10000);
CREATE PROCEDURE archive_in_batches(batch int)
LANGUAGE plpgsql AS $$
DECLARE
    n int;
BEGIN
    LOOP
        UPDATE big_log SET archived = true
        WHERE id IN (SELECT id FROM big_log WHERE NOT archived ORDER BY id LIMIT batch);
        GET DIAGNOSTICS n = ROW_COUNT;
        EXIT WHEN n = 0;
        COMMIT;   -- каждая порция — отдельная транзакция: короткие блокировки, меньше bloat
        RAISE NOTICE 'обработано % строк', n;
    END LOOP;
END $$;
CALL archive_in_batches(4000);
SELECT count(*) FILTER (WHERE archived) AS archived FROM big_log;

-- Триггер updated_at
CREATE TABLE articles (
    id         int PRIMARY KEY,
    title      text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT '2026-01-01 00:00+00'
);
CREATE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := '2026-11-06 12:00+00';   -- в реальном коде: now()
    RETURN NEW;
END $$;
CREATE TRIGGER articles_touch BEFORE UPDATE ON articles
FOR EACH ROW WHEN (OLD.* IS DISTINCT FROM NEW.*)   -- не трогать, если ничего не изменилось
EXECUTE FUNCTION touch_updated_at();
INSERT INTO articles VALUES (1, 'Черновик'), (2, 'Новость');
UPDATE articles SET title = 'Готово' WHERE id = 1;
UPDATE articles SET title = 'Новость' WHERE id = 2;   -- значение то же — триггер не сработает
SELECT * FROM articles ORDER BY id;

-- Аудит изменений в jsonb одной универсальной функцией
CREATE TABLE audit_log (id bigint GENERATED ALWAYS AS IDENTITY, tbl text, op text, old_row jsonb, new_row jsonb);
CREATE FUNCTION audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO audit_log (tbl, op, old_row, new_row)
    VALUES (TG_TABLE_NAME, TG_OP,
            CASE WHEN TG_OP <> 'INSERT' THEN to_jsonb(OLD) END,
            CASE WHEN TG_OP <> 'DELETE' THEN to_jsonb(NEW) END);
    RETURN NULL;   -- AFTER-триггер: результат игнорируется
END $$;
CREATE TRIGGER accounts_audit AFTER INSERT OR UPDATE OR DELETE ON accounts
FOR EACH ROW EXECUTE FUNCTION audit();
UPDATE accounts SET balance = balance + 1 WHERE id = 2;
DELETE FROM accounts WHERE id = 1;
SELECT op, old_row ->> 'balance' AS old_balance, new_row ->> 'balance' AS new_balance FROM audit_log ORDER BY id;

-- Категории изменчивости влияют на оптимизацию и индексы
CREATE FUNCTION label(id int) RETURNS text LANGUAGE plpgsql AS $$ BEGIN RETURN 'L' || id; END $$;
CREATE TABLE t (id int);
CREATE INDEX ON t (label(id));   -- VOLATILE по умолчанию — в индексе нельзя
ALTER FUNCTION label(int) IMMUTABLE;  -- результат зависит только от аргументов — честно помечаем
CREATE INDEX t_label_idx ON t (label(id));

-- SECURITY DEFINER: выполняется с правами владельца; search_path фиксируем обязательно
CREATE FUNCTION safe_balance(acc int) RETURNS numeric
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public
AS $$ SELECT balance FROM public.accounts WHERE id = acc $$;
SELECT proname, provolatile, prosecdef, proconfig FROM pg_proc
WHERE proname IN ('vat', 'transfer', 'rich_accounts', 'safe_balance') ORDER BY proname;
