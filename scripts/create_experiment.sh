#!/usr/bin/env bash
set -euo pipefail

# create_experiment.sh — отправляет POST /api/v1/experiments/create
# с файлами из testdata/ и тестовыми полями.
#
# Использование:
#   ./scripts/create_experiment.sh
#   URL=http://localhost:9999 ./scripts/create_experiment.sh

URL="${URL:-http://localhost:8091}/api/v1/experiments/create"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$(cd "$SCRIPT_DIR/../testdata" && pwd)"

echo "→ POST $URL"
echo "→ testdata: $TESTDATA_DIR"
echo ""

curl -k -v -X POST "$URL" \
  -F "title=Test Experiment" \
  -F "zenith_angle=45.5" \
  -F "latitude=43.1" \
  -F "longitude=131.9" \
  -F "comments=Created from test script" \
  -F "experiment_files=@$TESTDATA_DIR/archive.zip" \
  -F "background=@$TESTDATA_DIR/b2651321.051986" \
  -F "meteo=@$TESTDATA_DIR/meteo.csv"
