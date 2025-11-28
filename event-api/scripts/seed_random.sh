#!/usr/bin/env bash
# scripts/seed_random.sh
# Wrapper that calls the Python seeder which POSTS events to the API.

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PY_SCRIPT="$SCRIPT_DIR/seed_random_via_api.py"

if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is required but not found."
	exit 1
fi

echo "Seeding random events via API using $PY_SCRIPT"
python3 "$PY_SCRIPT" "$@"

echo "Done."
