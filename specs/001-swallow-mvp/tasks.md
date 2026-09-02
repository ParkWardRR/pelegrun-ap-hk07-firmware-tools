# Tasks 001 — actionable breakdown

**Stage:** Tasks · **Spec:** [spec.md](spec.md) · **Plan:** [plan.md](plan.md)
Legend: [x] done · [~] in progress · [ ] todo. Maps to `../../ROADMAP-DEV.md`.

## Phase 1 — Core + scaffold  (this release)
- [x] T1.1 Cargo workspace + `quarry` crate (lib + bin)
- [x] T1.2 Senao header parse/validate + one-field `product_id` re-head (`header.rs`)
- [x] T1.3 Code27 check-char + `make/validate_serial` + `model_code` (`serial.rs`)
- [x] T1.4 20-char `snextra` generate/validate
- [x] T1.5 `quarry` CLI: `inspect | rehead | serial | snextra | check`
- [x] T1.6 12 unit tests incl. hardware-verified vectors (EPC1X4200011 / snextra)
- [x] T1.7 Go module + `swallow` CLI skeleton (`version`, `plan`) building clean
- [x] T1.8 Falconry internal package stubs (eyas/jess/hood/band/mews/creance/lure)
- [x] T1.9 Zig `lure` stub building on 0.16
- [x] T1.10 Spec Kit artifacts (constitution, spec, plan, tasks) + CI + docs

## Phase 2 — Access & discovery (eyas, jess)  ✅
- [x] T2.1 `eyas` fingerprint (EWS-LuCI vs cloud-React vs FIT) over HTTP
- [x] T2.2 `jess` SSH:8822 adapter (legacy ssh-rsa) — exec channel
- [x] T2.3 `jess` cloud GUI API adapter (login → bearer → sys_info/force_ac/upload)
- [x] T2.4 `jess` LuCI adapter (md5(pw+\n) login, stok, flashops)
- [x] T2.5 Focused adapters (Cloud/LuCI/SSH) with a shared insecure client + LuciPassword

## Phase 3 — Safety, backup, provisioning (hood, mews, band)  ✅
- [x] T3.1 `hood` fw_printenv parse + completeness gate; append-only writer
- [x] T3.2 `mews` mtd7/8/11 dump + config export + hashes → dated bundle
- [x] T3.3 `band` unique serial + local inventory + collision preflight (quarry via WASM)

## Phase 4 — Safe flash + verify (no UART)
- [ ] T4.1 A/B slot detection
- [ ] T4.2 flash inactive slot (LuCI two-step / cloud upload+fw_upgrade)
- [ ] T4.3 reboot watch + env/serial re-verify + rollback

## Phase 5 — UART recovery (creance, lure)
- [ ] T5.1 `creance` serial console + scripted gated env repair
- [ ] T5.2 `lure` Zig TFTP/BOOTP responder (real)

## Phase 6 — Release polish
- [ ] T6.1 cross-compiled release binaries (mac/linux/win) + checksums
- [ ] T6.2 recorded-fixture adapter tests; property tests for quarry
- [ ] T6.3 user docs + field-guide cross-link
