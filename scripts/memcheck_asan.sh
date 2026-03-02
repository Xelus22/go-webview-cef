#!/usr/bin/env bash
set -euo pipefail

EXAMPLE="${1:-chromeless_mode}"

export CGO_CFLAGS="${CGO_CFLAGS:-} -O1 -g -fsanitize=address -fno-omit-frame-pointer"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -fsanitize=address"
export ASAN_OPTIONS="${ASAN_OPTIONS:-detect_leaks=1:halt_on_error=1:log_path=asan}"
export LSAN_OPTIONS="${LSAN_OPTIONS:-suppressions=}"

echo "[memcheck] building $EXAMPLE with ASan/LSan"
./run.sh build "$EXAMPLE" --rebuild

echo "[memcheck] running $EXAMPLE (close window to finish)"
./run.sh run "$EXAMPLE"
