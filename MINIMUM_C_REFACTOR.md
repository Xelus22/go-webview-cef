# Minimum-C Refactor Status

This document tracks the Go-first refactor that minimizes C usage while preserving existing behavior.

## Implemented

- Added `internal/cefruntime` as the Go-owned runtime core:
  - lifecycle (`Initialize`, `Run`, `Shutdown`),
  - browser registry/state tracking,
  - JS bridge dispatch to Go bindings,
  - window API surface used by `cef` and `adapter/webview`.
- Added `internal/cefshim` as the only runtime cgo boundary package:
  - tiny callback trampoline registration,
  - ABI wrappers for `cef_runtime_*` entrypoints,
  - callback forwarding into Go dispatcher.
- Added `internal/cefbindings` generated package:
  - `BrowserHandle` and `BrowserOptions` generated artifact,
  - deterministic regeneration script: `scripts/gen_cef_bindings.sh`.
- Refactored public `cef` package to a thin wrapper over `internal/cefruntime`.
- Removed duplicate legacy C-heavy paths:
  - `internal/cef/`,
  - `internal/cef_c/`,
  - `runtime/cef_runtime.c`.
- Kept one active C runtime implementation now located at:
  - `internal/cefshim/cef_runtime_impl.c`.
- Added local memory diagnostics scripts (non-CI gating):
  - `scripts/memcheck_asan.sh`,
  - `scripts/memcheck_valgrind.sh`.

## Ownership Contract

- Go runtime owns application state and binding dispatch.
- C shim owns only ABI callback bridges and direct CEF C API calls.
- C strings allocated by shim for one-shot calls are freed in the shim.
- Process argv allocated by shim for CEF init is owned by shim and freed on `Shutdown`.
- Browser handles are opaque (`BrowserHandle`) and only manipulated through runtime/shim APIs.

## Verification Commands

- `./run.sh build chromeless_mode --rebuild`
- `./run.sh build chromeless_mode_webview --rebuild`
- `timeout 20s ./run.sh run chromeless_mode --rebuild`
- `timeout 20s ./run.sh run chromeless_mode_webview --rebuild`

Both examples compile and enter the CEF message loop under this refactor.
