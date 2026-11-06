-- Первая таблица, первые строки, первые метакоманды psql.
SELECT current_setting('server_version_num')::int >= 180000 AS pg18_or_newer,
       current_user, current_database() LIKE 'course%' AS in_course_db;

CREATE TABLE books (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title     text NOT NULL,
    author    text NOT NULL,
    year      int  CHECK (year BETWEEN 1450 AND 2100),
    price_rub numeric(10, 2) NOT NULL DEFAULT 0
);

INSERT INTO books (title, author, year, price_rub) VALUES
    ('Мастер и Маргарита', 'Михаил Булгаков', 1967, 590),
    ('Пикник на обочине', 'Аркадий и Борис Стругацкие', 1972, 450),
    ('Designing Data-Intensive Applications', 'Martin Kleppmann', 2017, 3200);

SELECT id, title, year FROM books ORDER BY year;

-- \d — описание таблицы: столбцы, типы, ограничения, индексы
\d books

-- \x включает расширенный вывод: одна колонка на строку
\x on
SELECT * FROM books WHERE id = 1;
\x off

-- \dt — список таблиц, \di — индексов
\dt
\di
