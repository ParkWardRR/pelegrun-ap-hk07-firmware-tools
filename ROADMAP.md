# Roadmap

## What ships today (v0.5.0)

Two binaries: `quarry` (Rust) and `pelegrun` (Go).

| Tool | What it does |
|---|---|
| `quarry inspect` | Parse and display Senao image header (product_id, vendor_id, model, magic) |
| `quarry rehead` | Patch the `product_id` field so a sibling image passes another model's upload gate |
| `quarry serial` | Generate a valid 12-char Code27 serial for a given model code |
| `quarry snextra` | Generate the 20-char u-boot `snextra` (config field 19) |
| `quarry check` | Validate a serial's check character |
| `pelegrun discover` | Fingerprint an AP's firmware family (Cloud / EWS-LuCI / FIT) from its HTTP response |
| `pelegrun envcheck` | Gate on bootloader env completeness — refuses writes on a fragile env |
| `pelegrun redact` | Scrub passwords, tokens, MACs, and custom values from logs before sharing |
| `pelegrun serial/snextra/check` | Go wrappers around the same Code27 math as quarry |
| `pelegrun` (no args) | TUI dashboard showing the seven-step conversion sequence |

### Safety invariants

1. **Write the INACTIVE A/B slot.** The working slot stays bootable.
2. **Bootloader env is APPEND-ONLY.** Never erase, never rebuild from scratch.

These are enforced in `hood` (env gate) and documented in `SAFETY.md`.

---

## What's next

| Priority | Item | Status |
|---|---|---|
| **Now** | Test FIT v1.1.65-2 against EPC — if it works, the FIT path avoids synthetic serials entirely | Open |
| **Now** | Trigger EPC auto-firmware download for ECW230v3 (populate `firmware_config` in Mongo) | Open |
| Soon | Fleet inventory file + batch conversion planner (read-only, no auto-flash) | Planned |
| Later | OpenWrt install planning — see [ews377apv3-openwrt](https://github.com/ParkWardRR/ews377apv3-openwrt) | Moved |
| Later | Adapter contract for boards beyond ap-hk07 | Planned |

## Completed

| Phase | What shipped |
|---|---|
| Core math (quarry) | Header re-head, Code27 serial, property tests, real-image validation |
| Firmware fingerprint (eyas) | Detect Cloud / EWS-LuCI / FIT from HTTP response |
| Network access (jess) | SSH:8822, cloud GUI API, LuCI HTTP client |
| Env safety (hood) | Append-only gate, completeness check |
| Serial provisioning (band) | Unique serials, collision preflight |
| Backup data (mews) | MTD partition + config evidence bundle format |
| Secret scrubbing (redact) | Structural secrets + MAC + custom values |
| TUI (Bubble Tea) | Seven-screen dashboard with live package output |
| Release binaries | Cross-compiled Go (5 targets) + Rust (macOS ARM64), SHA256SUMS |

## Not planned (removed)

These were in earlier roadmap drafts but are out of scope for a tool used on a ~6 AP fleet:

- Fleet batch execution engine with resumable journals and canary orchestration
- FIT post-adoption proof framework (JSON-in/JSON-out validators)
- Hardware qualification matrix and support tier registry
- SBOM, release signatures, and provenance attestation
- Threat model document

The single-device `quarry rehead` + `pelegrun discover/envcheck` workflow is the right size for this use case. If fleet automation is ever needed, it belongs in the EPC controller or a separate ops tool, not in the firmware patcher.
