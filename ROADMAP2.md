# 🗺️ Roadmap

Where this toolkit is and where it is going.

**Direction:** macOS + OrbStack first → Linux-native next. Windows remains a reverse-engineering and capture source only; it is never a supported end-user flashing path.

**Operating principle:** preserve recoverability, prove every protocol assumption from captures, and make every destructive action explicit, validated, and reproducible.

**Legend:** ✅ done · 🚧 in progress · ⏳ planned · 💡 idea/nice-to-have

---

## Project guardrails

| Guardrail | Product decision |
|---|---|
| Destructive operations | `flash` must always identify the physical USB device, show the image hash/model compatibility result, create or confirm a backup, and require an interactive confirmation unless `--yes` is explicitly passed. |
| Firmware provenance | Never treat a filename, vendor marketing name, VID:PID alone, or an unverified community upload as sufficient identity. Record SHA-256, source, date, device descriptors, and validation status for every catalogued image. |
| Recovery | A documented manual recovery route must exist before native flashing is presented as stable. The tool should retain a last-known-good image reference and recognize bootloader-only state. |
| Protocol confidence | Capture at least two successful flash transactions with different image sizes or versions before treating fields such as length, checksum, block size, and address increment as understood. |
| Scope control | The flashing transport, image parser/validator, device fingerprinting, TUI, and EQ/control-channel work should remain independently testable modules. Do not let PEQ work block flashing safety or Linux support. |
| Privacy | `probe` submissions should be reviewable text/JSON, scrub host-specific serial data by default, and explain exactly what users are asked to attach to issues. |

---

## ✅ Done — through `v1.0.0`

| | Milestone | Notes |
|---|---|---|
| ✅ | **Reverse-engineer the normal-mode protocol** | `KT_USB_BOOT` HID transport recovered from the vendor tool with rizin + Ghidra → [`docs/EXTRACTION.md`](docs/EXTRACTION.md), [`docs/PROTOCOL.md`](docs/PROTOCOL.md) |
| ✅ | **Confirm a real cross-flash** | Moondrop KT02H20 (`31B2:0111`) → FiiO JadeAudio JA11 (`2972:0102`) |
| ✅ | **`ktflash` Rust tool** | Single Rust binary using `rusb`; `probe`, `handshake`, `unlock`, and `bootdiag` commands |
| ✅ | **Live TUI** | ratatui dashboard with device card, mode badge, stock → bootloader → JA11 journey, activity log, demo mode, and README GIF |
| ✅ | **Device classification** | Recognizes JA11 `2972:0102`, stock `31B2:0111`, other `31B2:*`, and bootloader `8888:CDC0` |
| ✅ | **macOS → OrbStack transport** | [`orbstack/ktflash-orbstack.sh`](orbstack/ktflash-orbstack.sh) reaches HID endpoints that macOS blocks directly |
| ✅ | **Core docs** | Flashing, compatibility, revert, protocol, extraction, and dongle-survey documentation |

---

## 🚧 In progress

### M1 — Reverse the CDC bootloader download protocol ⭐ critical path

The `T12345678` unlock reboots the dongle into a **CDC serial bootloader** (`8888:CDC0`). The vendor application drives it through a Qt `QSerialPort` state machine (`Shake hand → Erase → Program → UPGRADE FIRMWARE SUCCESS`). Its exact framing remains hidden in the Qt `QByteArray` machinery. Recovering this protocol unblocks fully native flashing.

#### Active path: ground-truth USB capture

1. Put a known-good dongle on a Windows capture machine with the vendor application and USBPcap.
2. Capture the appropriate root hub with `USBPcapCMD` while doing one controlled, known-safe update such as JA11 → the exact same JA11 image.
3. Extract bootloader bulk transfers from CDC endpoints `0x03` OUT and `0x83` IN with `tshark`.
4. Build a chronological transaction table: timestamp, direction, endpoint, payload hex, interpreted command, expected/observed reply, block index/address, and image-offset hypothesis.
5. Identify handshake, erase, program, finalize, verify/reboot, ACK/NAK, retry, timeout, block length, endian order, address stepping, and checksum/CRC fields.
6. Cross-check every candidate interpretation against Ghidra's send/parse functions (`FUN_00b8bfd0`, `FUN_00b8c380`) and the Qt state transitions.
7. Reimplement the minimum protocol in Rust behind a transport-independent trait, then validate it first against a no-write/replay fixture and only then on sacrificial hardware.

Example extraction command:

```sh
tshark -r cap.pcap -Y 'usb.transfer_type==0x03 && usb.endpoint_address in {0x03,0x83}' -T fields -e frame.time_epoch -e usb.endpoint_address -e usb.capdata
```

#### Faster paths worth using in parallel

| Tool / method | What it saves | Suggested use |
|---|---|---|
| **`usbrply`** | Avoids manually reconstructing every captured exchange by eye; it can turn Wireshark capture traffic into replay-oriented libusb/Python code. | Generate a replay reference from the first capture. Treat it as a behavioral oracle, not production code; port only the understood state machine to Rust. |
| **Frida or MinHook/API monitor injection** | Logs arguments passed to `QSerialPort::write()` and received from `readyRead()` before they become raw USB transfers. This is easier to correlate with Qt code than `usb.capdata`. | Hook the vendor process only on the dedicated capture PC. Log timestamps, thread ID, byte arrays, and call stack/module offsets. Compare buffer logs one-for-one against USBPcap. |
| **Ghidra headless / `ghidra-headless-mcp`** | Removes repeated GUI work when finding strings, xrefs, callers, callee graphs, and field offsets inside the vendor executable. | Export a stable function map and decompiler snapshots as research artifacts. Use headless scripts to search `Shake hand`, `Erase`, `Program`, `SUCCESS`, `QByteArray`, and `QSerialPort` call sites. |
| **`socat -x` virtual-serial logging** | Gives a human-readable hex log if the software can be redirected through a virtual COM pair; useful confirmation if USBPcap dissections are noisy or incomplete. | Use as a secondary observation route, not as the canonical capture. Preserve raw USBPcap capture regardless. |
| **Protocol differential testing** | Separates invariant frame fields from image-dependent fields. | Capture at least two successful upgrades, ideally with different firmware sizes/versions. Diff the sequences block-by-block and inspect changed byte positions. |

#### M1 evidence requirements

| Requirement | Definition of done |
|---|---|
| Capture quality | At least one complete successful flash, from bootloader enumeration through success/reboot, with no unexplained missing bulk transfers. |
| Framing | Header, command ID, payload-length encoding, checksum/CRC placement and endianness either demonstrated or clearly marked unknown. |
| Data transfer | Confirmed maximum payload/block size, address or block-counter behavior, ACK semantics, retries, and terminal-image behavior. |
| Safety | A deliberately invalid image is rejected locally before write; no experiment relies on sending arbitrary frames to a production dongle. |
| Fixture | A sanitized capture-derived fixture and parser tests cover every observed command/reply class. |
| Documentation | `docs/BOOTLOADER_PROTOCOL.md` names observed facts separately from hypotheses and includes capture hashes and tool versions. |

**Status:** dongle staged on Windows ✅ · `tshark` ready on macOS ✅ · USBPcap driver install in progress 🚧 · capture + decode ⏳.

---

## ⏳ Next milestones

| | Milestone | Definition of done | Depends on |
|---|---|---|---|
| ⏳ | **M2 · Native `ktflash flash <fw.bin>`** | Unlock → wait for bootloader → handshake → erase → program → finalize → verify/reboot. Includes `KT_Helios` magic validation, strict file-length bounds, SHA-256 display, checksum/CRC verification, target-device compatibility checks, `--dry-run`, explicit confirmation, timeout/retry policy, and machine-readable JSON result. | M1 |
| ⏳ | **M2.1 · Flash-safety UX** | Default `flash` workflow offers/creates a backup first, shows a compact preflight report, requires a physical-device match after re-enumeration, and refuses ambiguous multi-dongle situations. `--yes` exists only for scripted use and must require explicit device/image selectors. | M2 |
| ⏳ | **M3 · Linux-native + Linux release** | Same Rust binary uses native Linux HID/serial access; a packaged `99-ktflash.rules` udev rule allows non-root operation; test on Debian/Ubuntu and Arch; release signed/checksummed x86_64 and aarch64 binaries. | M2 |
| ⏳ | **M4 · Backup stock firmware before replacement** ⭐ | `ktflash dump` reads and verifies a running image where protocol capability permits. `flash` defaults to backup-first. Backup metadata records device fingerprint, descriptor snapshot, firmware hash, timestamp, tool version, and validation result. | M1 / protocol capability confirmation |
| ⏳ | **M5 · JCALLY JM12 first-class support** | Fingerprint a real JM12, preserve a verified clean stock dump, add a model-scoped revert command, and document all uncertainty. Start with a community issue template/request before buying additional units. | M4 |
| ⏳ | **M6 · Compatibility matrix from real data** | Public matrix uses a controlled schema: exact product/board markings, VID:PID before and after, descriptors, firmware hash, stock/target version, host OS/kernel, success result, audio behavior, and recovery result. “Reported” and “maintainer-verified” are distinct statuses. | M2.1 |
| ⏳ | **M7 · Dongle discovery / supported-device detector** | `probe` identifies positive, negative, and uncertain candidates without claiming compatibility solely from VID:PID. Candidate flow produces a shareable sanitized report and asks for no destructive action. | M6 |
| ⏳ | **M8 · Brick-recovery path** | A documented manual recovery playbook exists before stable native flashing: stuck-bootloader detection, expected VID:PID, re-enumeration timeout, last-known-good image selection, known limitations, and when Windows vendor recovery remains necessary. Native automatic recovery is added only after it is repeatably validated. | M2 + M4 |
| ⏳ | **M9 · CI, release, and supply-chain hardening** | Pull requests run format, lint, unit tests, fixture tests, documentation-link check, dependency audit, and dependency policy checks. Tags build reproducible macOS/Linux artifacts and publish checksums, SBOM/provenance where practical, and release notes. | Now; should start before M3 |
| ⏳ | **M10 · Regression hardware harness** | At least one sacrificial supported dongle plus USB switching/controlled host workflow validates unlock/enumeration/flash/revert cycles. If full hardware CI is impractical, maintain capture-based integration fixtures and a documented manual release checklist. | M2 |

---

## Implementation architecture to add

### Transport and protocol boundaries

| Boundary | Responsibility | Test strategy |
|---|---|---|
| `device` | Enumerate, identify, open, claim/release interfaces, wait for re-enumeration, record descriptors | Mock enumerator plus saved descriptor fixtures |
| `hid_control` | Existing normal-mode HID handshake/unlock/control traffic | Golden request/response vectors |
| `bootloader` | CDC framing, command state machine, retry/timeout logic, erase/program/finalize/verify | Capture-derived transcript tests and fake serial transport |
| `image` | Parse header/magic, length limits, checksum/CRC, SHA-256, compatibility metadata | Corpus of valid, truncated, corrupt, and wrong-family images |
| `backup_catalog` | Sidecar metadata, integrity checks, provenance, lookup of safe recovery candidates | Temp-directory tests; never silently overwrite backups |
| `cli` / `tui` | Human confirmation, JSON output, no-TUI scripting, useful errors | Snapshot tests for preflight and failure messages |

This separation is important: raw USB transport is difficult to test in CI, while byte-framing, image validation, and state transitions should be highly testable without a physical dongle.

### Recommended command surface

```text
ktflash probe [--json] [--verbose]
ktflash handshake [--json]
ktflash unlock [--wait-bootloader]
ktflash bootdiag [--json]
ktflash inspect <firmware.bin> [--json]
ktflash dump --output <backup.bin> [--metadata <backup.json>]
ktflash flash <firmware.bin> [--backup auto|require|skip] [--dry-run] [--device <selector>] [--yes]
ktflash recover [--image <backup.bin>|--latest-compatible] [--dry-run] [--yes]
ktflash catalog verify <image-or-directory>
```

`inspect` should land before `flash`: users and issue reporters need a harmless way to establish whether an image appears structurally valid and what metadata/hash it carries.

### Exit codes and automation

| Code | Meaning |
|---:|---|
| 0 | Operation completed and verification passed |
| 2 | Invalid CLI input or incompatible image/device preflight failure |
| 3 | Permission, transport, or enumeration failure |
| 4 | Device entered bootloader but did not complete expected protocol state |
| 5 | Flash began but device-side verification/reboot failed; preserve recovery details prominently |
| 6 | User declined confirmation / dry run completed successfully |

Provide `--json` early. It makes support bundles, CI-style capture analysis, and advanced user automation possible without scraping the TUI.

---

## Firmware catalog and backup policy

### Do not commit arbitrary firmware blobs by default

A Git repository containing vendor firmware can create redistribution, provenance, and trust problems. Prefer a metadata catalog first. Store images only when redistribution is clearly permitted or users explicitly provide them with the relevant rights understood.

| Field | Why it belongs in every image/backup sidecar |
|---|---|
| SHA-256 | Canonical identity; prevents confusion from renamed files |
| File size | Quick corruption/truncation check |
| Format/magic and parser version | Explains why the tool accepted or rejected it |
| Source URL or “user dump” provenance | Makes trust and rediscovery auditable |
| Device fingerprint | Includes VID:PID, bcdDevice, manufacturer/product strings, interface layout, and safe non-identifying board details if known |
| Firmware version/build fields | Useful only if extracted/observed reliably; otherwise explicitly `unknown` |
| Capture/tool version | Associates an image with the protocol implementation used to validate it |
| Validation status | `observed`, `structurally_valid`, `flash_verified`, `audio_verified`, `maintainer_verified`, or `rejected` |
| License/redistribution note | Avoids accidental publication of material that should only be locally backed up |

Example sidecar structure:

```json
{
  "schema_version": 1,
  "sha256": "<64 lowercase hex characters>",
  "size_bytes": 0,
  "origin": {
    "kind": "local_dump",
    "source": "user-provided",
    "captured_at": "2026-09-04T00:00:00Z"
  },
  "device": {
    "vid": "31b2",
    "pid": "0111",
    "bcd_device": "0000",
    "product": "unknown",
    "fingerprint_version": 1
  },
  "image": {
    "magic": "KT_Helios",
    "declared_version": "unknown",
    "validation": "structurally_valid"
  },
  "tool": {
    "ktflash_version": "1.0.0",
    "protocol_revision": "unknown"
  }
}
```

---

## Compatibility-matrix rules

| Status | Meaning | User-facing wording |
|---|---|---|
| Candidate | Shares a superficial identifier or physical similarity only | “Not tested; do not flash based on this entry.” |
| Fingerprinted | `probe` observed compatible-looking descriptors/protocol behavior | “Recognized candidate; flashing compatibility is unconfirmed.” |
| Backup verified | A stock dump was captured and passed structural/integrity checks | “Backup path established; target flash still unverified.” |
| Flash verified | Named source image completed native flash and the device re-enumerated | “Flash transaction passed; audio/function checks may still be incomplete.” |
| Function verified | Flash verified plus audio/controls tested on a recorded host/device setup | “Known working under recorded conditions.” |
| Revert verified | Stock backup or supported revert image restored successfully | “Recovery path tested.” |
| Unsupported / rejected | Probe or image validation indicates incompatible behavior | “Do not attempt flashing.” |

Never conflate “same DAC chip,” “same case,” “same vendor app,” “same VID,” or “same bootloader VID:PID” with a safe cross-flash. Those are discovery signals, not compatibility proofs.

---

## M9 — CI and release plan

### Pull-request quality gate

```sh
cargo fmt --check && cargo clippy --all-targets --all-features -- -D warnings && cargo test --all-features --locked && cargo audit && cargo deny check
```

| Gate | Purpose |
|---|---|
| `cargo fmt --check` | Keeps contributor changes mechanically consistent |
| `cargo clippy -- -D warnings` | Catches Rust mistakes and prevents warning debt |
| `cargo test --locked` | Tests against the committed, reproducible dependency set |
| Capture/image fixture tests | Prevents a protocol-parser change from silently changing interpretation of known traffic |
| `cargo audit` | Checks `Cargo.lock` against RustSec advisories |
| `cargo deny check` | Adds policy checks for advisories, licenses, duplicate crates, and allowed dependency sources |
| Docs/link check | Prevents release instructions and research links from rotting |

### Release artifacts

| Artifact | Minimum contents |
|---|---|
| macOS universal or per-architecture archive | Binary, `README`/quick start, SHA-256 checksum, version, signing/notarization status |
| Linux x86_64 archive | Binary, udev rule, install/uninstall instructions, SHA-256 checksum |
| Linux aarch64 archive | Same as x86_64; important for SBC/lab users and ARM Linux desktops |
| Source archive | Git tag, generated changelog, license, `Cargo.lock` |
| SBOM/provenance | Nice-to-have at first, required if the project begins receiving broad adoption or enterprise use |

Use `cargo-dist` if it fits the desired packaging model; otherwise use `cross-rs`/`cargo-zigbuild` plus a small explicit GitHub Actions matrix. Prefer the boring option that produces deterministic artifacts and is easy for contributors to maintain.

### Versioning policy

| Change type | Suggested release meaning |
|---|---|
| Patch | Documentation, diagnostics, non-protocol bug fixes, no behavioral flashing change |
| Minor | New device fingerprint, non-breaking command, validated capability, new supported-image metadata |
| Major | Changed CLI automation contract, changed backup format, altered protocol behavior, or any release that changes safety/default destructive behavior |

For this project, protocol/image support deserves prominently curated release notes; a conventional changelog alone is not enough. Add a small “Newly verified / newly rejected / recovery changes” section per release.

---

## 💡 Future capability ideas

| Idea | Why it is valuable | Do not start until |
|---|---|---|
| **EQ / PEQ over `0xFF01`** | Lets the tool become a useful control application, not only a flasher; can expose filter, gain, and PEQ safely | M2/M8 are stable; control changes must have a read-back/default-reset strategy |
| **TUI actions** | Makes the polished dashboard actionable | CLI flashing preflight and error semantics are mature; TUI must not hide irreversible details |
| **Auto-recovery** | Detect stuck bootloader and offer the last compatible known-good image | M8 has a validated manual recovery story and a safe image catalog |
| **Homebrew tap** | Makes macOS installation straightforward | Stable macOS artifact, clear OrbStack dependency story, and support capacity |
| **Notarized macOS binary** | Reduces Gatekeeper friction | Active Apple Developer membership, signing identities, secure CI secret handling, and a repeatable notarization workflow |
| **Device support bundles** | One command creates sanitized probe/descriptors/logs/capture metadata for GitHub issues | Privacy scrubber and clear consent text exist |
| **Bench/diagnostic mode** | Repeatedly enumerate, unlock, and collect timing data without flashing | Hardware harness or clear safeguards exist |
| **Formal protocol specification** | Easier independent implementations and long-term maintenance | M1 has multiple captures and unresolved hypotheses are clearly separated |

---

## Documentation to add

| File | Purpose |
|---|---|
| `CONTRIBUTING.md` | Rust setup, local test command, demo mode, fixture policy, how to submit a probe report, and what hardware experiments are acceptable |
| `SECURITY.md` | Vulnerability-reporting contact/process; clarify that firmware and device recovery issues may be safety-sensitive |
| `docs/BOOTLOADER_PROTOCOL.md` | Capture-grounded CDC protocol spec: facts vs hypotheses, state diagram, frames, timing, retries, and capture hashes |
| `docs/RECOVERY.md` | Manual recovery first, then native recovery. Include expected bootloader identity and exact stop/escalation guidance |
| `docs/FIRMWARE_POLICY.md` | Hash/provenance convention, redistribution stance, backup retention, and image validation criteria |
| `docs/COMPATIBILITY_SCHEMA.md` | Defines matrix statuses and required submission fields |
| `.github/ISSUE_TEMPLATE/device-report.yml` | Structured collection of descriptors, firmware hash, host OS, test result, and consent for sharing logs |
| `.github/ISSUE_TEMPLATE/firmware-dump.yml` | Structured clean-stock-dump request with provenance and no encouragement to redistribute restricted vendor firmware |

---

## How to help

The highest-leverage contribution remains **M1**: a clean, complete serial-bootloader capture from a known-safe vendor-tool upgrade. See [`docs/EXTRACTION.md`](docs/EXTRACTION.md) §5 and [`research/`](research/).

High-value non-destructive contributions:

| Contribution | What to attach |
|---|---|
| Candidate-dongle fingerprint | `ktflash probe --json`, USB descriptors before/after normal operation, product/board markings, and where purchased |
| Known-good stock backup | Hash plus metadata sidecar; share image bytes only where permitted and desired |
| Before/after behavior report | VID:PID, descriptors, operating system, audio behavior, button/mic behavior, and revert result |
| M1 capture review | Sanitized pcap/capture-derived transaction table, tool versions, vendor-app version, and clearly labeled hypotheses |
| CI/documentation contribution | A small focused PR with local test results; no hardware required |

Do **not** ask contributors to flash an unverified target image merely to populate the matrix. A useful matrix includes negative and uncertain outcomes, but evidence must not be collected by encouraging avoidable device loss.

---

## External tools and references

These are implementation accelerators, not project dependencies by default.

| Tool | Use in this project | Primary link |
|---|---|---|
| USBPcap | Windows-side USB capture for M1 | [USBPcap](https://desowin.org/usbpcap/) |
| Wireshark / `tshark` | Decode, filter, and export USB capture data | [Wireshark](https://www.wireshark.org/docs/man-pages/tshark.html) |
| `usbrply` | Generate replay-oriented code from Wireshark USB captures | [GitHub: JohnDMcMaster/usbrply](https://github.com/JohnDMcMaster/usbrply) |
| Ghidra | Static analysis, xrefs, decompilation, headless automation | [Ghidra](https://github.com/NationalSecurityAgency/ghidra) |
| Frida | Optional dynamic instrumentation of vendor-app serial calls | [Frida](https://frida.re/) |
| `win-api-monitor` | Example MinHook/DLL-injection approach for Windows userland call monitoring | [GitHub: jayo78/win-api-monitor](https://github.com/jayo78/win-api-monitor) |
| `socat` | Optional serial relay and hex-dump observation | [socat](http://www.dest-unreach.org/socat/) |
| `cargo-dist` | Release artifact/install-script/changelog automation option | [GitHub: axodotdev/cargo-dist](https://github.com/axodotdev/cargo-dist) |
| `cross-rs` | Linux cross-compilation option for Rust | [GitHub: cross-rs/cross](https://github.com/cross-rs/cross) |
| `cargo-audit` | RustSec advisory checks for `Cargo.lock` | [GitHub: RustSec/rustsec](https://github.com/RustSec/rustsec/tree/main/cargo-audit) |
| `cargo-deny` | License, source, advisory, and dependency-policy checks | [GitHub: EmbarkStudios/cargo-deny](https://github.com/EmbarkStudios/cargo-deny) |

---

## Recommended sequencing

1. **Finish M1 with redundant evidence:** USBPcap capture plus static Qt-state-machine correlation; add dynamic QSerialPort hook only if raw capture remains ambiguous.
2. **Land `inspect` and image-validation primitives before `flash`:** it immediately provides value and makes the destructive path smaller and safer.
3. **Implement M2 as a library-first state machine with capture fixtures:** resist putting protocol logic directly in the TUI/CLI command handler.
4. **Write recovery documentation and backup metadata alongside M2/M4, not after:** this is product safety, not polish.
5. **Start M9 now:** CI and fixture infrastructure will make protocol iteration safer long before Linux releases.
6. **Ship M3 after one stable native flash/revert story:** use prebuilt x86_64/aarch64 binaries and clear udev installation instructions.
7. **Use M6/M7 to expand only from evidence:** first-class support should require a fingerprint, a known image/provenance path, a tested flash, and ideally a tested revert.
8. **Defer EQ/TUI flash controls until transport safety is boring:** the roadmap's differentiation is reliable cross-platform recovery-capable flashing, not feature count.
