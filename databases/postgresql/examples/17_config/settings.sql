-- Откуда берётся значение параметра и когда изменение вступает в силу.
-- boot_val — значение, вшитое в сервер; фактическое значение показывает SHOW или setting.
SELECT name, context, unit, boot_val
FROM pg_settings
WHERE name IN ('shared_buffers', 'work_mem', 'maintenance_work_mem', 'effective_cache_size',
               'max_connections', 'random_page_cost', 'effective_io_concurrency', 'io_method', 'io_workers',
               'max_wal_size', 'checkpoint_timeout', 'wal_level', 'synchronous_commit', 'jit')
ORDER BY context, name;

-- Иерархия: значение по умолчанию → postgresql.conf → ALTER SYSTEM → база → роль → сессия → транзакция
SELECT format('ALTER DATABASE %I SET work_mem = %L', current_database(), '32MB') \gexec
\c
SELECT current_setting('work_mem') AS work_mem, source FROM pg_settings WHERE name = 'work_mem';

SET work_mem = '64MB';
SELECT current_setting('work_mem') AS work_mem, source FROM pg_settings WHERE name = 'work_mem';

BEGIN;
SET LOCAL work_mem = '256MB';          -- только до конца транзакции: удобно для тяжёлого отчёта
SELECT current_setting('work_mem') AS in_transaction;
COMMIT;
SELECT current_setting('work_mem') AS after_commit;

-- Параметр уровня postmaster нельзя поменять в сессии
SET shared_buffers = '1GB';
