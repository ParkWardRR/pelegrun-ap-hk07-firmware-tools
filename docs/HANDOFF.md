# Handoff — fleet control plane, on-device dump, FIT/adapter gates

This note is for the agent picking up after this session to test and fix against
**real ap-hk07 hardware**. Everything below is pure decision/plan logic, fully
unit-tested with no device I/O. None of it has touched real hardware yet. Your job
is to wire the accessors, run against devices, and correct the assumptions flagged
here. Preserve the safety invariants — they are the point of the project.

## Toolchain gotcha (read first)

`go/go.mod` pins `go 1.27`. This session's sandbox could only run **go 1.26.1**, so
every change was validated by temporarily lowering the directive to `go 1.26`,
then restoring `go 1.27` before each commit. The code uses no 1.27-only features.
Your environment has `GOTOOLCHAIN=auto`, so `make ci` will auto-fetch 1.27 — just
run it normally. If you ever need to iterate on 1.26, lower the directive locally
and restore it before committing.

## CI / releases are LOCAL only

There is **no hosted CI** — the GitHub Actions workflows were removed on request.

- `make ci` is the gate (fmt-check + lint + race tests across Rust/Go/Zig).
- `make hooks` installs `githooks/pre-push`, which runs `make ci` before any push.
  Run it once in your clone. Bypass in an emergency with `git push --no-verify`.
- Releases are built locally: `make dist` (`scripts/dist.sh`).

## What landed this session

All new code is Go, in `go/internal/…`, matching the existing plan-emitting,
IO-free style (decide WHAT to do; an accessor does the I/O).

### 1. `internal/fleet` — P9 control plane (biggest piece)
- `inventory.go` — normalized `Device` identity model: stable physical identity
  (model/serial/MAC + confidence) vs mutable software/access/evidence. JSON
  load/save, canonical hashing, staleness.
- `reconcile.go` — fold a fresh discovery reading into a record; a changed
  model/serial or a fully disjoint MAC set is an **identity conflict** (confidence
  → conflict), never a silent update.
- `policy.go` — deterministic per-device eligibility (model/family/confidence/
  staleness/backup), maintenance window, error budget. Hashed so a changed policy
  invalidates a built plan.
- `plan.go` — `BuildPlan` (read-only, deterministic, changes nothing) with
  per-device identity pins; `ApplyGuard` fails closed on a tampered plan, changed
  policy/target, closed window, missing device, identity mismatch, stale reading,
  or lost eligibility.
- `canary.go` — reproducible cohort selection (count/percent, model-diverse) and a
  promote/hold/abort health gate bounded by the error budget.
- `journal.go` — durable per-device operation state machine (mirrors the ROADMAP
  lifecycle table); legal-transition enforcement, terminal states.
- `executor.go` — **`Accessor` interface** the executor drives:
  `Preflight / Backup / WriteInactive / VerifyWrite / Reboot / Validate`.
  Persists intent BEFORE each mutation; `Resume` refuses to blindly repeat a
  half-applied write/reboot.

### 1b. `internal/fleetexec` — SSHAccessor (the Accessor wiring, started)
`SSHAccessor` implements `fleet.Accessor` over an injected command `Runner`
(`jess.SSH.Run` matches it). The **read-only** steps are implemented and tested:
Preflight/Validate confirm the env is complete (hood) + optional serial; Backup
runs the on-device `dump` plan and returns the SHA-256 of the device's
`SHA256SUMS`, refusing if a recovery-critical artifact is missing. The
**destructive** steps are REQUIRED injected hooks (`WriteInactiveFn`,
`VerifyWriteFn`, optional `PointBootFn`/`HealthFn`) because the write path is
firmware-family-specific — a nil hook errors, never silently no-ops. Remaining
hardware work: fill those hooks per family (cloud/LuCI/FIT), add a
`CloudAccessor` sibling for the HTTP families, pull dump artifacts to host and
call `dump.Manifest.Verify`, and confirm where the real serial actually lives.
See the NOTES block at the top of `internal/fleetexec/ssh_accessor.go`.

### 2. `internal/fitadopt` — P10a FIT real-serial eligibility
Read-only gate: FIT ≥ 1.1.65, FIT family, 64-hex image digest + provenance, ≥1
recovery route, and a **real device-read serial** (Code27-checked; rejects
empty/generated/spoofed). Absolute rule: never generate or spoof a serial.

### 3. `internal/adapter` — P12a capability contract + P12d registry
Typed capabilities + support tiers. `CanFlash` requires ≥ experimental tier plus
`backup` + `flash_ab` + a recovery route; `Validate` catches tier/capability
incoherence. `Registry` is the machine-readable model DB (load/save/validate),
seeded with `ap-hk07` at `experimental`. Promote to `verified` only after the
ROADMAP hardware qualification matrix passes. CLI: `swallow adapters list|validate`.

### 3b. `internal/redact` — secret scrubbing (safety)
Scrubs passwords/tokens/private-keys/(opt-in)MACs and explicit literals
(serials, hostnames) from support bundles, fixtures, and logs. Run every shared
artifact through it: `swallow redact bundle.txt [--mac] [--value <serial>]`.
Verify on real fixtures that nothing sensitive survives before publishing.

### 4. `internal/dump` — on-device verified full-flash capture (NEW safety feature)
`swallow dump plan --dest /tmp/swallow-dump` prints a read-only dd/nanddump script
that captures every mtd partition to device-local storage and hashes it, plus a
provenance manifest. Safety net so a complete known-good image exists locally
before any flash. **See the big `NOTES FOR THE NEXT AGENT` block at the top of
`internal/dump/dump.go`** — the important hardware caveats live there:
- ap-hk07 is NAND → prefer `nanddump` (`--nand` / `PlanOptions{NAND:true}`);
  confirm the device actually has `nanddump`. Plain `dd` misses OOB and can choke
  on bad blocks.
- mtd numbering is not stable across firmware — always parse a live `/proc/mtd`.
- `--dest` must be OFF the NAND being dumped (tmpfs/USB). `SafeDestHint` warns; the
  accessor must confirm the real mount + free space from the `df` gate.
- `Verify()` compares on-device vs pulled-to-host hashes; also add a re-read hash
  compare on hardware.

## New CLI surface

```
swallow fleet plan  --inventory inv.json --policy policy.json --image <n> --image-sha256 <hex> [--out plan.json]
swallow fleet apply --inventory inv.json --policy policy.json --plan plan.json --image <n> --image-sha256 <hex> [--max-age 10m]
swallow dump  plan  --dest /tmp/swallow-dump [--proc-mtd file|-] [--nand] [--manifest out.json]
swallow redact [file|-] [--mac] [--value <serial>]...
swallow adapters list|validate [--file registry.json]
```

`fleet apply` currently stops at `ApplyGuard` (no accessor wired) and changes
nothing. `dump plan` is plan-only.

## Suggested order for the hardware pass

1. Implement `fleet.Accessor` over `jess` + the `flash` plan; run `fleet apply`
   end-to-end on ONE sacrificial device in a canary cohort of 1.
2. Wire `dump` to an accessor: run the script, pull artifacts, `Manifest.Verify`.
   Confirm dd vs nanddump and real `/proc/mtd` sizes/names on the bench.
3. Validate `fitadopt` version/serial assumptions against a real FIT device.
4. Fill the real ap-hk07 `adapter.Support` record (tier `experimental` until the
   qualification matrix in the ROADMAP passes).

## Safety invariants to keep

- Flash only ever writes the INACTIVE slot (`internal/flash`).
- env writes are append-only on a complete env (`internal/hood`).
- backup/evidence is a hard gate before any destructive step (`mews`, and the
  fleet policy `require_backup`).
- `dump` and `mews` never emit a command that writes an mtd.
- `fleet apply` fails closed on any drift; a `--yes`-style path must never become
  an unbounded destructive loop.
