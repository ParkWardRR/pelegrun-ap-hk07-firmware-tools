# Spec 002 — EPC pairing: flash + adopt ap-hk07 APs into a self-hosted EnGenius Private Cloud

**Stage:** Specify · **Status:** draft — framework only, no implementation yet
**Constitution:** all clauses apply (esp. II, III, VI, **VII — no secrets, no leaks**)

## Problem

Operators retiring EnGenius's legacy `ezMaster` controller (EOL, cleartext
`admin`/`password` by default) can stand up **EnGenius Private Cloud (EPC)** —
EnGenius's current on-prem controller, self-hostable as a container stack — and
manage their ap-hk07 fleet from it instead. Two gaps stand between "the AP is on
genuine, current firmware" (spec 001's territory) and "the AP is a stable, proven
member of the EPC's inventory":

1. **Cross-flash to a controller-capable firmware family** is already solved by
   `pelegrun crossflash` (product_id re-head + HTTP upload/upgrade) — this spec
   does not redo that work, it builds on top of it.
2. **Pairing/adoption into EPC itself** — pointing the AP at the controller,
   getting the controller to accept its checkin, and *proving* the pairing is
   real and durable — is currently a hand-driven, per-device, HTTP-request-level
   process with no tool support and no repeatable verification story. That's
   exactly the gap P10 already closed for FitController-style adoption
   (`fitadopt`); EPC needs the equivalent.

This spec defines the **framework** for that capability — package shape,
interfaces, CLI surface, and the proof model — for a follow-on implementation
pass. It intentionally does not lock in every protocol byte, because parts of
the EPC-side mechanism are still being reverse-engineered empirically against
real hardware as of this writing (see "Open questions").

## Users

Home-lab / small-fleet operators who have already cross-flashed their ap-hk07
APs to a controller-capable firmware family (cloud/ECW230v3, or FIT) and want to
self-host fleet management via EPC instead of vendor cloud or the EOL ezMaster.

## Primary scenarios

1. **Point an AP at a controller.** Given an AP already on controller-capable
   firmware, set its discovery override to a specific controller address and
   confirm the setting persists across a reboot.
2. **Register a device in controller inventory.** Given the AP's real,
   device-read identity (serial + MAC — never generated, per the same real-serial
   rule P10 established), add it to the controller's inventory under an
   operator-specified organization/network scope.
3. **Prove adoption, don't infer it.** Walk the same staged-evidence model P10
   already uses for FIT (`fitadopt.Prove`): identity agreement, controller
   pointer persistence, controller reachability, authentication success (not a
   silent/ambiguous failure), inventory match, and adopted status — captured as
   discrete, independently-checkable facts, not a single boolean.
4. **Survive real-world disruption.** An adopted device must stay adopted across
   AP reboots and controller restarts; a plan/verify step should make "does it
   survive both" an explicit, repeatable check, not a one-time observation.
5. **Recover from a stuck controller-side gate.** Some controller-side settings
   that gate device checkin have been observed (during exploratory testing) to
   silently revert to a disabled state — the plan/apply model needs to make that
   kind of precondition drift visible and re-assertable, not something an
   operator has to remember to babysit by hand.

## Functional requirements

- **FR1 (identity, read-only):** Read a device's real serial and MAC from the
  AP itself (reuse `jess.Cloud.SysInfo`; never generate/spoof — same absolute
  rule as `fitadopt`).
- **FR2 (point at controller):** Set and verify a discovery-override / controller
  address on the AP (reuse `jess.Cloud.SetForceAC`, already implemented; extend
  only if the apply/persist step needs its own verification beyond what exists).
- **FR3 (controller client):** A new package (working name `epcadopt`, mirroring
  `fitadopt`'s naming) that can authenticate to an EPC controller's own
  management surface and perform, at minimum: read inventory/device state,
  register a device under an operator-specified org/network scope, and read back
  checkin/connection status for a specific device. The controller's exact API
  shape (REST endpoints vs. requiring direct datastore access) is an **open
  question** — see below; the interface should be defined so the underlying
  transport is swappable without changing callers.
- **FR4 (eligibility gate, mirroring `fitadopt.Validate`):** Before attempting
  registration, validate: device model/family is one the target controller
  version is confirmed to support; a real (non-placeholder) serial and MAC are
  present; the AP has at least one recovery route recorded (same
  `RecoveryRoutes` shape `fitadopt.Request` already uses). Read-only, no I/O,
  deterministic — same shape as `fitadopt.Validate`.
- **FR5 (proof gate, mirroring `fitadopt.Prove`):** After attempting adoption,
  independently verify each stage in the primary-scenario-3 list above and
  report every failure found, not just the first. A "looks adopted" state is not
  a passing result until it has survived at least one AP reboot and one
  controller restart (or the plan explicitly documents why that check was
  skipped).
- **FR6 (plan/apply, not a single blocking call):** Mirror the project's
  existing `flash.Plan...`/`fleet` plan-then-apply shape: produce an ordered,
  human-readable list of steps (some gated) rather than a single opaque
  "adopt()" call, so an operator can see exactly what will happen before
  anything mutates controller or device state.
- **FR7 (CLI surface):** `pelegrun epc check|plan|prove`, mirroring the existing
  `fit check|prove` and `openwrt check|plan` command shapes exactly (same flag
  conventions, same read-only-unless-gated posture).
- **FR8 (no secrets in the tool or its config schema):** Every identity a
  command needs — controller address, org/network scope, credentials — is
  **operator-supplied input** (flags, env vars, or a local config file the
  operator owns), never a hardcoded default beyond well-known, publicly
  documented placeholders (e.g. a controller's documented first-run default
  credentials, if any — treat like `admin`/`admin` is already treated elsewhere
  in this codebase: flagged, not silently assumed safe).

## Non-goals

- Standing up or configuring an EPC controller instance itself (that's
  infrastructure/deployment, out of scope for this tool).
- A generic client for every EPC feature (RADIUS, mDNS discovery, VLAN/SSID
  provisioning) — this spec is scoped to the pairing/adoption lifecycle only.
- Automating discovery of *which* controller to use — the operator supplies the
  address explicitly (same posture as `SetForceAC` today).
- Resolving the still-open protocol questions below — that's this spec's
  intended next step, not something to guess at here.

## Open questions (resolve empirically before/during implementation)

1. **Does the controller expose a stable device-registration API**, or does
   reliable registration currently require direct datastore manipulation? If
   the latter is the only path today, FR3's "controller client" needs an
   explicit, clearly-labeled escape hatch for that — and the plan/apply model
   (FR6) must treat it as a distinctly higher-risk step than an API call.
2. **What is the controller's own web-session/API auth model** (cookie, bearer
   token, CSRF) — does it match the pattern `jess.Cloud` already implements for
   AP-side auth, or does it need its own client shape entirely?
3. **Which firmware families does the target controller version actually
   support for adoption** — this determines FR4's eligibility gate contents.
   Don't assume a specific version number without checking a real controller
   instance's own behavior (see the corrected `fitadopt.MinFitVersion` mistake
   in this repo's history for exactly why: a plausible-sounding version-support
   claim that was never actually checked against the vendor's own artifacts).
4. **Is there a documented, stable device-checkin protocol** (headers, auth
   scheme, timing/replay constraints) this tool can rely on, or is it
   undocumented/reverse-engineered and therefore something the tool must treat
   as best-effort and versioned defensively?
5. **What does "stays adopted" require operationally** — does controller
   config drift (a setting reverting on its own, as noted in scenario 4) reflect
   a controller bug, a controller restart side-effect, or an intentional
   security posture? This changes whether FR6's plan should re-assert that
   setting as a routine step or flag it as an anomaly to report.

## Acceptance criteria (for the framework itself, this spec)

- [ ] `specs/002-epc-pairing/{spec,plan,tasks}.md` exist and are internally
      consistent with each other and with `ROADMAP.md`.
- [ ] A package skeleton exists (interfaces + doc comments, no real transport
      logic) that a follow-on implementation pass can fill in without
      redesigning the shape.
- [ ] Nothing in the repo — code, tests, docs, or examples — contains a real
      controller IP/hostname, org/network ID, device serial/MAC, or credential
      (Constitution VII). All examples use clearly-fake placeholder values.
