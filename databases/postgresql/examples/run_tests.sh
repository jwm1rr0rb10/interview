#!/usr/bin/env bash
# Прогоняет все примеры курса и сверяет вывод psql с эталоном, как pg_regress.
#   bash run_tests.sh            — проверить все примеры
#   bash run_tests.sh 08         — только модуль 08
#   bash run_tests.sh --update   — перезаписать эталоны (после осознанных изменений)
set -uo pipefail
cd "$(dirname "$0")"

UPDATE=0; FILTER=""
for a in "$@"; do
  case "$a" in
    --update) UPDATE=1 ;;
    *) FILTER="$a" ;;
  esac
done

export PGUSER="${PGUSER:-postgres}"
# Одинаковые сообщения, часовой пояс и форматы на любой машине
export PGOPTIONS="-c lc_messages=C -c timezone=UTC -c datestyle=ISO,DMY -c intervalstyle=postgres -c extra_float_digits=1"

pass=0; fail=0
for f in [0-9][0-9]_*/*.sql; do
  [[ -n "$FILTER" && "$f" != "$FILTER"* ]] && continue
  dir=$(dirname "$f"); name=$(basename "$f" .sql)
  db="course_$(echo "${dir}_${name}" | tr -c 'a-z0-9_\n' '_')"
  exp="$dir/expected/$name.out"
  dropdb --if-exists "$db" >/dev/null 2>&1
  # builtin-провайдер локали (PG17+): одинаковая сортировка строк на любой ОС
  createdb -T template0 -E UTF8 --locale-provider=builtin --builtin-locale=C.UTF-8 "$db" || { echo "не удалось создать базу $db"; exit 1; }
  out=$(mktemp)
  # stdin, а не -f: в сообщениях об ошибках не будет пути к файлу
  psql -X -a -q -d "$db" < "$f" > "$out" 2>&1
  dropdb "$db"
  if [[ $UPDATE == 1 ]]; then
    mkdir -p "$dir/expected"; cp "$out" "$exp"; echo "обновлён $exp"; pass=$((pass+1))
  elif diff -u "$exp" "$out" > "$out.diff" 2>&1; then
    echo "ok    $f"; pass=$((pass+1))
  else
    echo "FAIL  $f"; head -40 "$out.diff"; fail=$((fail+1))
  fi
  rm -f "$out" "$out.diff"
done
echo "итого: $pass ok, $fail fail"
[[ $fail == 0 ]]
