-- Продажи по дням: ровные данные без случайности
CREATE TABLE sales (day date, region text, amount int);
INSERT INTO sales
SELECT d::date, r, 100 + (extract(day FROM d)::int * 37 + length(r) * 11) % 90
FROM generate_series('2026-11-01'::date, '2026-11-07', '1 day') d,
     unnest(ARRAY['Север', 'Юг']) r;

-- Агрегат сворачивает строки, оконная функция — нет: у каждой строки своё окно
SELECT day, region, amount,
       sum(amount) OVER (PARTITION BY region ORDER BY day) AS running_total,
       round(avg(amount) OVER (PARTITION BY region ORDER BY day
                               ROWS BETWEEN 2 PRECEDING AND CURRENT ROW), 1) AS avg_3d,
       amount - lag(amount) OVER (PARTITION BY region ORDER BY day) AS vs_prev_day
FROM sales
WHERE region = 'Север'
ORDER BY day;

-- Ранжирование: row_number, rank, dense_rank на одинаковых значениях
CREATE TABLE scores (player text, points int);
INSERT INTO scores VALUES ('Аня', 90), ('Боря', 85), ('Вика', 90), ('Гоша', 70), ('Даша', 85);
SELECT player, points,
       row_number() OVER w AS row_number,
       rank()       OVER w AS rank,
       dense_rank() OVER w AS dense_rank,
       round(percent_rank() OVER w::numeric, 2) AS pct_rank,
       ntile(2)     OVER w AS half
FROM scores
WINDOW w AS (ORDER BY points DESC)
ORDER BY points DESC, player;

-- Лучший день в каждом регионе: окно в подзапросе, фильтр снаружи
SELECT region, day, amount
FROM (
    SELECT *, row_number() OVER (PARTITION BY region ORDER BY amount DESC, day) AS rn
    FROM sales
) t
WHERE rn = 1
ORDER BY region;

-- Доля от итога и первое/последнее значение в окне
SELECT region, sum(amount) AS total,
       round(100.0 * sum(amount) / sum(sum(amount)) OVER (), 1) AS share_pct
FROM sales GROUP BY region ORDER BY region;

SELECT day, amount,
       first_value(amount) OVER w AS first_in_week,
       last_value(amount)  OVER w AS last_in_week
FROM sales WHERE region = 'Юг'
WINDOW w AS (ORDER BY day ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
ORDER BY day;

-- Перцентили и медиана — упорядоченные агрегаты
SELECT region,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY amount) AS median,
       percentile_disc(0.9) WITHIN GROUP (ORDER BY amount) AS p90,
       mode() WITHIN GROUP (ORDER BY amount % 10) AS most_common_last_digit
FROM sales GROUP BY region ORDER BY region;

-- GROUPING SETS / ROLLUP: несколько уровней итогов одним запросом
SELECT coalesce(region, 'ИТОГО') AS region,
       coalesce(to_char(day, 'YYYY-MM-DD'), 'все дни') AS day,
       sum(amount) AS total
FROM sales
WHERE day <= '2026-11-02'
GROUP BY ROLLUP (region, day)
ORDER BY region NULLS LAST, day NULLS LAST;

-- Заполнение пропусков: дни без продаж должны быть нулями
CREATE TABLE visits (day date, cnt int);
INSERT INTO visits VALUES ('2026-11-01', 5), ('2026-11-03', 8), ('2026-11-06', 2);
SELECT d::date AS day, coalesce(v.cnt, 0) AS cnt
FROM generate_series('2026-11-01'::date, '2026-11-06', '1 day') d
LEFT JOIN visits v ON v.day = d::date
ORDER BY 1;

-- Острова и пропуски: непрерывные серии дней с посещениями
CREATE TABLE logins (user_id int, day date);
INSERT INTO logins VALUES (1, '2026-11-01'), (1, '2026-11-02'), (1, '2026-11-03'),
                          (1, '2026-11-05'), (1, '2026-11-06'), (2, '2026-11-02');
SELECT user_id, min(day) AS streak_start, max(day) AS streak_end, count(*) AS days
FROM (
    SELECT user_id, day,
           day - (row_number() OVER (PARTITION BY user_id ORDER BY day))::int AS grp
    FROM logins
) t
GROUP BY user_id, grp
ORDER BY user_id, streak_start;
