#!/usr/bin/env bash
set -euo pipefail

EXAMPLE="${1:-chromeless_mode}"

./run.sh build "$EXAMPLE" --rebuild

export LD_LIBRARY_PATH="$(pwd)/build:${LD_LIBRARY_PATH:-}"

cd build

echo "[memcheck] valgrind $EXAMPLE (close window to finish)"
valgrind \
  --leak-check=full \
  --show-leak-kinds=all \
  --track-origins=yes \
  --error-exitcode=1 \
  "./$EXAMPLE"
