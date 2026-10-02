# Handoff — scope reduction notes

This note documents the v0.5.0 scope reduction. The Go CLI was slimmed from 16
internal packages and 12 subcommands down to 8 packages and 7 commands. The Zig
lure binary was removed. The TUI was kept.

## What was removed (and why)

The following were removed because they solved problems that don't exist yet at
the current 6-AP fleet scale:

| Cut component | Reason |
|---|---|
| `fleet` (P9) | 400+ lines, plan-only, "not wired yet" |
| `fleetexec` | Empty accessor wiring for future fleet execution |
| `fitadopt` (P10) | JSON-in/JSON-out validator, no real adoption |
| `adapter` (P12) | Registry for boards that don't exist yet |
| `dump` | Generates shell scripts you'd write yourself |
| `creance` | UART recovery — separate concern |
| `flash` | Only used by cut packages |
| `lure` (Zig) | TFTP recovery — separate binary, separate language |
| `tui-harness` | termwright E2E screenshot harness |

## What was kept

| Component | Role |
|---|---|
| `quarry` (Rust) | The core: header re-head, Code27 serial, snextra, inspect, UBI |
| `pelegrun` TUI | Bubble Tea dashboard — the seven-step guided flow |
| `pelegrun discover` | Fingerprint firmware family via `eyas` |
| `pelegrun envcheck` | Bootloader env completeness gate via `hood` |
| `pelegrun redact` | Secret scrubbing via `redact` |
| `pelegrun serial/snextra/check` | Serial utilities via `band` |
| `mews` | Backup/evidence bundle data (used by TUI) |
| `jess` | HTTP client used by `discover` |
| `tuigif` | Pure-Go tour.gif + screenshot renderer |

## Recovering the removed code

All removed code is preserved in the git history before the v0.5.0 tag. The full
P9/P10/P12 designs, prototypes, and test suites can be restored from there if and
when real fleet needs materialise.

## Toolchain note

`go/go.mod` pins `go 1.27`. Your environment has `GOTOOLCHAIN=auto`, so
`make ci` will auto-fetch 1.27 — just run it normally.

## CI / releases are LOCAL only

There is **no hosted CI** — no GitHub Actions.

- `make ci` is the gate (fmt-check + lint + race tests across Rust + Go).
- `make hooks` installs `githooks/pre-push`, which runs `make ci` before any push.
- Releases are built locally: `make dist` (`scripts/dist.sh`).
