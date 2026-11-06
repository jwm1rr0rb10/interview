-- NULL — «неизвестно»: трёхзначная логика и её ловушки.
SELECT NULL = NULL AS eq, NULL <> 1 AS ne, NULL IS NULL AS is_null,
       NULL IS DISTINCT FROM NULL AS distinct_nulls,
       1 IS DISTINCT FROM NULL AS distinct_1_null;

SELECT true AND NULL AS and_t, false AND NULL AS and_f,
       true OR NULL AS or_t, NOT NULL::boolean AS not_null;

CREATE TABLE employees (name text, manager text, bonus int);
INSERT INTO employees VALUES ('Аня', NULL, 100), ('Боря', 'Аня', NULL), ('Вика', 'Аня', 50);

-- WHERE отбрасывает строки, где условие NULL: Аня не попадёт ни в одну выборку
SELECT name FROM employees WHERE manager = 'Аня';
SELECT name FROM employees WHERE manager <> 'Аня';
SELECT name FROM employees WHERE manager IS DISTINCT FROM 'Аня';

-- Агрегаты пропускают NULL; count(*) считает строки, count(col) — непустые значения
SELECT count(*) AS rows, count(bonus) AS with_bonus, sum(bonus), avg(bonus) FROM employees;

-- COALESCE — первое не-NULL значение; NULLIF — защита от деления на ноль
SELECT name, coalesce(bonus, 0) AS bonus, 100 / nullif(coalesce(bonus, 0), 0) AS ratio
FROM employees ORDER BY name;

-- Главная ловушка: NOT IN с NULL в подзапросе не возвращает ничего
SELECT name FROM employees WHERE name NOT IN (SELECT manager FROM employees);
-- NOT EXISTS работает как ожидается
SELECT e.name FROM employees e
WHERE NOT EXISTS (SELECT 1 FROM employees s WHERE s.manager = e.name);

-- Сортировка: NULL по умолчанию последние при ASC, первые при DESC
SELECT name, bonus FROM employees ORDER BY bonus DESC;
SELECT name, bonus FROM employees ORDER BY bonus DESC NULLS LAST;
