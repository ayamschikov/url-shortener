#!/usr/bin/env bash
#
# Генератор нагрузки для наблюдения за метриками в Grafana.
# Сначала создаёт несколько коротких ссылок, затем льёт смешанный трафик:
#   ~70% GET существующего кода  → 301 redirect (после прогрева — cache hit)
#   ~20% GET случайного кода     → 404 (cache miss → промах в БД)
#   ~10% POST /shorten           → 201 (создание)
#
# Параметры через env: BASE, DURATION (сек), CONCURRENCY, NUM_URLS.
# Пример: DURATION=180 CONCURRENCY=12 ./monitoring/loadtest.sh
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
DURATION="${DURATION:-120}"
CONCURRENCY="${CONCURRENCY:-8}"
NUM_URLS="${NUM_URLS:-20}"

echo "Seeding ${NUM_URLS} short URLs at ${BASE} ..."
codes=()
for i in $(seq 1 "$NUM_URLS"); do
  code=$(curl -s -X POST "$BASE/shorten" \
    -H 'Content-Type: application/json' \
    -d "{\"url\":\"https://example.com/page/$i\"}" \
    | sed -n 's/.*"short_url":"\([^"]*\)".*/\1/p')
  [ -n "$code" ] && codes+=("$code")
done

if [ "${#codes[@]}" -eq 0 ]; then
  echo "ERROR: no codes seeded — is the app running on ${BASE}? (make run)"
  exit 1
fi
echo "Seeded ${#codes[@]} codes. Generating load: ${CONCURRENCY} workers for ${DURATION}s ..."

worker() {
  local end=$(( $(date +%s) + DURATION ))
  while [ "$(date +%s)" -lt "$end" ]; do
    r=$((RANDOM % 10))
    if [ "$r" -lt 7 ]; then
      c=${codes[$((RANDOM % ${#codes[@]}))]}
      curl -s -o /dev/null "$BASE/$c"
    elif [ "$r" -lt 9 ]; then
      curl -s -o /dev/null "$BASE/zzz$RANDOM"
    else
      curl -s -o /dev/null -X POST "$BASE/shorten" \
        -H 'Content-Type: application/json' \
        -d "{\"url\":\"https://example.com/r/$RANDOM\"}"
    fi
  done
}

for _ in $(seq 1 "$CONCURRENCY"); do worker & done
wait
echo "Done."
