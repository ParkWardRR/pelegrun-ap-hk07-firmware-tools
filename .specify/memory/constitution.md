# Constitution — pelegrun-ap-hk07-firmware-tools

Non-negotiable principles. Every spec, plan, task, and PR is checked against these.

## I. No brick by design (the two invariants)
1. **Write the INACTIVE A/B slot; keep the active one bootable.** A failed flash
   must always leave a working slot reachable by a reset-button hold.
2. **The bootloader env is APPEND-ONLY.** Only ever `fw_setenv <field>` on an env
   first verified COMPLETE. Never erase, never hand-rebuild, never expose a
   `setconfig -a 5`-equivalent. A valid-but-incomplete env is the one thing that
   bricks — the tool must be structurally incapable of producing one.

## II. Prove, don't assume
Every mutating step is followed by verification (env re-read, serial confirm,
reboot watch). Absence of an error is not proof of success.

## III. Backup before you touch
Per-device evidence bundle (mtd7/8/11 dumps, config export, firmware hashes)
is a hard gate before any destructive step. ART is read-only-backup, never write.

## IV. Pure core, guarded edges
Deterministic byte/string logic lives in the Rust `quarry` core with exhaustive
tests. Hardware/network/serial I/O lives behind gated adapters. The core never
does I/O.

## V. Language discipline
Rust for pure/safety-critical math; Go for orchestration, SSH/HTTP, UART; Zig for
the freestanding recovery responder. Each where it is strongest — not dogma.

## VI. Legal & ethical scope
Unofficial; not affiliated with, endorsed by, or supported by EnGenius/Senao.
"EnGenius" and "Senao" are used nominatively to name the affected products. For
**interoperability and self-hosting on hardware you own**. Never for warranty
fraud, evading paid licensing on hardware you do not own, or defeating theft
protection. No vendor firmware images are redistributed by this repo. Blue Oak 1.0.0.

## VII. No secrets, no leaks
No IPs, hostnames, real serials/MACs, controller org/DB IDs, or credentials ever
land in the repo. The tool ships generic logic; users supply their own inputs.
