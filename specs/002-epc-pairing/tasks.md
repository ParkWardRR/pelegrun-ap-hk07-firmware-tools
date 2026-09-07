# Tasks 002 — actionable breakdown

**Stage:** Tasks · **Spec:** [spec.md](spec.md) · **Plan:** [plan.md](plan.md)
Legend: [x] done · [~] in progress · [ ] todo · **(live)** requires a reachable
real controller/AP to complete, not just source review.

## Phase 0 — Protocol reconnaissance (resolve spec.md's open questions)
- [ ] T0.1 **(live)** Confirm the controller's own web-session/API auth model
      (cookie vs bearer vs CSRF) against a real instance.
- [ ] T0.2 **(live)** Confirm whether device registration has a stable API, or
      requires direct datastore access — and if the latter, exactly which
      operations are unavoidable that way.
- [ ] T0.3 **(live)** Confirm which firmware families/versions the target
      controller version actually accepts for adoption — do not carry over an
      unverified version-support claim (see the `fitadopt.MinFitVersion`
      correction elsewhere in this repo's history as the cautionary precedent).
- [ ] T0.4 **(live)** Confirm the device-checkin protocol's stability/versioning
      posture (is it documented anywhere, or purely reverse-engineered).
- [ ] T0.5 Write up T0.1–T0.4's findings **generically** (no real IPs/IDs/MACs)
      as a follow-up note in this spec directory, updating spec.md's "Open
      questions" section with resolved answers before Phase 2 begins.

## Phase 1 — Package skeleton (no live controller needed)
- [x] T1.1 `go/internal/epcadopt/` package created with doc comment
      cross-referencing this spec.
- [x] T1.2 `Request`/`Result`/`Validate` types stubbed (signatures + doc
      comments only), mirroring `fitadopt.Request`/`Result`/`Validate`.
- [x] T1.3 `Expected`/`Observed`/`Prove` types stubbed, mirroring
      `fitadopt.Expected`/`Observed`/`Prove`.
- [x] T1.4 `Client` interface stubbed per plan.md (narrow: Login /
      RegisterDevice / DeviceStatus) — no concrete implementation yet.
- [ ] T1.5 Fill in `Validate`/`Prove` logic once T0.3's findings are in
      (depends on Phase 0).

## Phase 2 — Controller client implementation
- [ ] T2.1 Concrete `Client` implementation for whichever transport T0.1/T0.2
      confirmed (HTTP API client, or the datastore-seeder escape hatch from
      plan.md, or both if the real mechanism is a hybrid).
- [ ] T2.2 `httptest`-based unit tests with synthetic fixtures (no real
      captured traffic — regenerate request/response shapes from the
      confirmed protocol, don't paste real payloads that might carry real
      IDs).
- [ ] T2.3 If a datastore-seeder path is needed: an explicit safety comment on
      why it's a distinct, higher-risk interface (per plan.md), plus whatever
      minimal precondition checks are feasible without a live connection.

## Phase 3 — CLI
- [ ] T3.1 `go/cmd/pelegrun/epc.go`: `check` subcommand (read-only identity +
      reachability + auth report), modeled on `cmdFitCheck`/`cmdOpenWrtCheck`.
- [ ] T3.2 `plan` subcommand: build and print the ordered, gated plan (reusing
      `flash.Step`), modeled on `cmdOpenWrtPlan`.
- [ ] T3.3 `prove` subcommand: modeled on `cmdFitProve` exactly (same
      `--expected`/`--observed` JSON-file pattern).
- [ ] T3.4 Wire into `main.go`'s dispatch + usage text, same pattern as the
      `openwrt`/`crossflash` additions.
- [ ] T3.5 `runCap`-based CLI tests: usage, missing required flags, and (with
      synthetic `httptest` servers only) the check/plan happy paths — mirror
      `openwrt_test.go`/`crossflash_test.go` structure closely.

## Phase 4 — Registry/adapter integration
- [ ] T4.1 Decide whether EPC-adoptability is a new `adapter.Capability`
      (e.g. `CapEPCAdopt`) alongside the existing `CapFlashAB`/
      `CapFlashFixedPartition` — if so, add it following the exact pattern
      `CapFlashFixedPartition` used (new const, doc comment explaining the
      distinct promise, `AllCapabilities` update, `Validate`/`CanFlash`
      interaction considered).
- [ ] T4.2 Decide whether an EPC-adopted registry record needs its own
      `Support` entry (mirroring how `ap-hk07-openwrt` got a distinct `Model`
      string from `ap-hk07`) or whether adoption status belongs in a separate,
      non-`adapter`-package model entirely (adoption is fleet/instance state,
      not a firmware capability — lean toward the latter unless a concrete
      need for the former turns up during implementation).

## Phase 5 — Docs + evidence (**live** for the actual walkthrough content)
- [ ] T5.1 **(live)** `docs/USAGE.md` section: EPC pairing walkthrough, written
      the way the existing "HTTP-only cross-flash" section is — real command
      sequences, generic placeholder values only.
- [ ] T5.2 **(live)** Extend `README.md`'s command examples block, matching the
      `crossflash`/`openwrt` entries' style.
- [ ] T5.3 **(live)** ROADMAP.md: promote this from "framework only" to a
      tracked phase (P13, or fold into P10 if T0.3 finds EPC and FitController
      share enough mechanism to unify) once Phase 0–3 are done and validated
      against a real controller at least once.

## Explicit non-tasks (per spec.md's non-goals)
- Deploying/configuring an EPC controller instance.
- A general-purpose EPC client covering RADIUS/mDNS/VLAN/SSID features.
- Automating controller discovery.
