# Plan 001 — technical plan

**Stage:** Plan · **Status:** approved → Tasks · **Spec:** [spec.md](spec.md)

## Architecture (falconry pipeline)

```mermaid
flowchart LR
  U([operator]) --> S[swallow · Go]
  S --> E[eyas · discover/fingerprint]
  E --> J[jess · access: ssh8822 / cloud / luci / uart]
  J --> M[mews · backup mtd7/8/11 + config]
  M --> H[hood · env-completeness gate]
  H --> Q[quarry · Rust: re-head + serial]
  Q --> B[band · provision unique serial]
  B --> F[flash · inactive A/B slot]
  F --> V[verify · reboot + re-read + confirm]
  H -. fragile? .-> X[[refuse / stop]]
  J -. dead board .-> C[creance · UART gated env repair]
  C --> L[lure · Zig: TFTP re-flash]
```

## Tech stack + rationale
- **Rust `quarry` (done):** pure header/serial math, exhaustively unit-tested.
  Zero I/O. Ships as a lib + a standalone CLI so it is useful today.
- **Go `swallow`:** orchestration + all I/O. `x/crypto/ssh` is the only SSH stack
  that cleanly does the legacy `HostKeyAlgorithms=+ssh-rsa` these APs need on
  :8822; `net/http`+InsecureSkipVerify for the cloud GUI (`-k`) and LuCI. Trivial
  static cross-compile; `go.bug.st/serial` for UART. Rust core reached via a WASM
  module (wazero) or subprocess — no cgo.
- **Zig `lure`:** freestanding TFTP/BOOTP responder for deep-brick recovery,
  embedded in the binary, no runtime.

## Safety model (maps to Constitution I/II/III)
- `hood` parses `fw_printenv`, asserts `bootcmd`/`active_fw`/`rootfsname` present
  and coherent **before** any write; writes are append-only `fw_setenv`. No API in
  the codebase can erase or partial-save the env.
- `flash` always targets the inactive slot and re-reads slot state after.
- `mews` runs first and aborts the pipeline if the backup bundle is incomplete.

## Testing & CI
- Rust: `cargo test` (12 tests today) + property tests later; the header/serial
  vectors are pinned to hardware-verified values.
- Go: `go vet` + `go build` per-OS; adapter logic tested against recorded fixtures
  (no live device in CI).
- Zig: `zig build`.
- CI matrix runs all three on push.

## Milestones → dev phases
See [tasks.md](tasks.md) and `../../ROADMAP-DEV.md`. MVP = FR1–FR6 (dev phases 1–4).
