-- Полнотекстовый поиск: лексемы вместо подстрок
SELECT to_tsvector('russian', 'Быстрые коты бегали по крышам') AS tsvector;
SELECT to_tsvector('russian', 'Коты бегают') @@ to_tsquery('russian', 'кот & бегать') AS matches;

CREATE TABLE articles (
    id    int PRIMARY KEY,
    title text NOT NULL,
    body  text NOT NULL,
    -- генерируемая колонка: заголовок важнее текста (вес A против B)
    tsv   tsvector GENERATED ALWAYS AS (
              setweight(to_tsvector('russian', title), 'A') ||
              setweight(to_tsvector('russian', body), 'B')) STORED
);
CREATE INDEX articles_tsv_idx ON articles USING gin (tsv);
INSERT INTO articles (id, title, body) VALUES
 (1, 'Индексы в PostgreSQL', 'B-tree, GIN и BRIN: какой индекс выбрать для поиска.'),
 (2, 'Транзакции и блокировки', 'Как избежать дедлоков при параллельных обновлениях.'),
 (3, 'Полнотекстовый поиск', 'Поиск по словам с учётом морфологии: индексы GIN и ранжирование.'),
 (4, 'Репликация', 'Физическая и логическая репликация, переключение на реплику.');

-- websearch_to_tsquery понимает синтаксис поисковиков: кавычки, OR, минус
SELECT id, title, round(ts_rank(tsv, q)::numeric, 3) AS rank
FROM articles, websearch_to_tsquery('russian', 'индекс поиск') AS q
WHERE tsv @@ q
ORDER BY rank DESC, id;

-- Внимание: минус относится к соседнему слову, AND связывает сильнее OR:
-- 'репликация OR дедлок -логическая' = реплик | (дедлок & !логическ)
SELECT websearch_to_tsquery('russian', 'репликация OR дедлок -логическая') AS parsed;
SELECT id, title FROM articles
WHERE tsv @@ websearch_to_tsquery('russian', 'репликация OR дедлок -логическая') ORDER BY id;

-- Подсветка найденного
SELECT ts_headline('russian', body, websearch_to_tsquery('russian', 'индексы'),
                   'StartSel=<b>, StopSel=</b>') AS snippet
FROM articles WHERE id = 3;

-- Префиксный поиск для автодополнения
SELECT title FROM articles WHERE tsv @@ to_tsquery('russian', 'реплик:*') ORDER BY id;

-- На 4 строках Seq Scan дешевле индекса; отключим его ненадолго, чтобы увидеть план с GIN
EXPLAIN (COSTS OFF) SELECT id FROM articles WHERE tsv @@ websearch_to_tsquery('russian', 'индекс');
SET enable_seqscan = off;
EXPLAIN (COSTS OFF) SELECT id FROM articles WHERE tsv @@ websearch_to_tsquery('russian', 'индекс');
RESET enable_seqscan;

-- pg_trgm: нечёткий поиск по опечаткам и подстрокам
CREATE EXTENSION pg_trgm;
CREATE TABLE cities (name text);
INSERT INTO cities VALUES ('Москва'), ('Санкт-Петербург'), ('Новосибирск'), ('Екатеринбург'), ('Нижний Новгород');
CREATE INDEX cities_trgm_idx ON cities USING gin (name gin_trgm_ops);

SELECT name, round(similarity(name, 'Новосибирк')::numeric, 2) AS sim
FROM cities WHERE name % 'Новосибирк'           -- % — «похоже» (порог pg_trgm.similarity_threshold = 0.3)
ORDER BY sim DESC;

SELECT name FROM cities WHERE name ILIKE '%бург%' ORDER BY name;

-- KNN: ближайшие по похожести — GiST-индекс умеет сортировать по расстоянию <->
CREATE INDEX cities_trgm_gist ON cities USING gist (name gist_trgm_ops);
SELECT name, round((name <-> 'Нижний')::numeric, 2) AS distance FROM cities ORDER BY name <-> 'Нижний' LIMIT 2;
