# Spec 001 — Pelegrún MVP: no-UART cross-flash + serial provisioning

**Stage:** Specify · **Status:** implemented — MVP (FR1–FR6) shipped in v0.4.0; FR7–FR9 also landed
**Constitution:** all clauses apply (esp. I, II, VI).

## Problem
The documented EWS377AP v3 → FIT/cloud "bridge" firmware is EOL and unavailable.
Cross-flashing the sibling ap-hk07 images by hand works, but the manual path is a
minefield: wrong `setconfig` usage wipes the bootloader env, a hand-rebuilt env
bricks the boot slot, SSH is on a non-obvious port, and adoption fails silently on
a blank serial. Users need a tool that makes the safe path the only path — and
that recovers a device **without UART wherever physically possible**.

## Users
- Home-lab / small-fleet operators retiring ezMaster for a self-hosted controller.
- Anyone with ap-hk07-family hardware (EWS377AP v3 / EWS377-FIT / ECW230v3) they own.

## Primary scenarios
1. **Convert & provision (no UART).** Point the tool at an AP; it fingerprints the
   firmware, backs up flash, patches a supplied image's `product_id`, provisions a
   unique serial (append-only), flashes the inactive slot, and verifies.
2. **Repair a stuck env (no UART, if the OS still shells).** Detect an incomplete
   env; restore missing fields append-only; verify.
3. **Recover a dead board (UART).** When no network shell exists: drive the u-boot
   console through a *gated* `env default -a → inspect → env save`, or TFTP a
   fresh image if the rootfs is gone.

## Functional requirements
- **FR1 (quarry, ✅):** parse/validate the Senao header; one-field `product_id`
  re-head; Code27 serial + 20-char `snextra` generate/validate. Pure, tested —
  including property tests and validation against real vendor images.
- **FR2 (eyas, ✅):** fingerprint firmware family (EWS-LuCI / cloud-React / FIT).
- **FR3 (jess, ✅):** access adapters — SSH:8822, cloud GUI API, LuCI (UART via
  creance) — one interface.
- **FR4 (mews, ✅):** per-device backup bundle (mtd7/8/11 + config + hashes) as a
  hard gate.
- **FR5 (hood, ✅):** env-completeness gate; append-only writes; refuse fragile
  states.
- **FR6 (flash, ✅):** A/B-aware flash (inactive slot), with verification +
  rollback.
- **FR7 (band, ✅):** serial uniqueness + collision preflight against a local
  inventory.
- **FR8 (creance + lure, ✅):** UART console recovery (gated env repair; TFTP
  deep-brick re-flash).
- **FR9 (✅):** single static binary per OS; no runtime deps — cross-compiled
  release binaries + checksums ship per tag.

## Non-goals
- No bundled vendor firmware. No controller-side seeding of DBs. No support for
  defeating licensing/theft protection. No cloud accounts.

## Acceptance criteria (MVP = FR1–FR6) — met in v0.4.0
- ✅ `quarry` re-head + serial math match hardware-verified values; tests green in
  CI, plus property tests and real-image validation (6 product ids across ~26
  genuine images), and a Go↔Rust parity check on the serial math.
- ✅ The tool **cannot** issue an env-erase or a partial-env-save (enforced in
  `hood`; no `PlanErase`/`PlanReset` exists, empty values are refused).
- ✅ A failed flash always leaves a bootable slot (`flash` targets the inactive
  slot; unit-tested).
- ✅ Every mutating command has a paired verify step and a captured evidence
  artifact (`flash.Plan` ordering; `mews` bundle gate).
