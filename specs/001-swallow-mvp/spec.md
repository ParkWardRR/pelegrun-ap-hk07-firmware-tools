# Spec 001 — swallow MVP: no-UART cross-flash + serial provisioning

**Stage:** Specify · **Status:** approved → Plan
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
- **FR1 (quarry, done):** parse/validate the Senao header; one-field `product_id`
  re-head; Code27 serial + 20-char `snextra` generate/validate. Pure, tested.
- **FR2:** fingerprint firmware family (EWS-LuCI / cloud-React / FIT).
- **FR3:** access adapters — SSH:8822, cloud GUI API, LuCI, UART — one interface.
- **FR4:** per-device backup bundle (mtd7/8/11 + config + hashes) as a hard gate.
- **FR5:** env-completeness gate; append-only writes; refuse fragile states.
- **FR6:** A/B-aware flash (inactive slot), with verification + rollback.
- **FR7:** serial uniqueness + collision preflight against a local inventory.
- **FR8:** UART console recovery (gated env repair; TFTP deep-brick re-flash).
- **FR9:** single static binary per OS; no runtime deps.

## Non-goals
- No bundled vendor firmware. No controller-side seeding of DBs. No support for
  defeating licensing/theft protection. No cloud accounts.

## Acceptance criteria (MVP = FR1–FR6)
- `quarry` re-head + serial math match hardware-verified values; tests green in CI.
- The tool **cannot** issue an env-erase or a partial-env-save (enforced in code).
- A failed flash always leaves a bootable slot (documented + tested against fixtures).
- Every mutating command has a paired verify step and a captured evidence artifact.
