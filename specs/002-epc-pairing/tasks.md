# Tasks 002 — actionable breakdown

**Stage:** Tasks · **Spec:** [spec.md](spec.md) · **Plan:** [plan.md](plan.md)
Legend: [x] done · [~] in progress · [ ] todo · **(live)** requires a reachable
real controller/AP to complete, not just source review.

## Phase 0 — Protocol reconnaissance (resolve spec.md's open questions)
- [x] T0.1 **(live)** Confirmed 2026-09-07 via read-only introspection against
      a real controller instance (imported its own FastAPI app module,
      called its `.openapi()`, and probed auth behavior with empty/garbage
      payloads via an in-process TestClient — no real network calls, no
      writes, no state mutated). Result: NOT a single cookie/CSRF model —
      user-facing endpoints use a custom JWT-bearer decorator (clean `401
      Not authenticated`); device checkin uses a separate header-based
      scheme entirely (see T0.4). See spec.md's Open Questions §1/§2.
      **Partially re-opened same day, second probe:** the one route shaped
      like a login endpoint (`POST /api/v1/jwt-token`) is itself gated behind
      the same JWT-bearer decorator — a missing Authorization header and an
      invalid-but-present bearer token return two DIFFERENT 401 bodies
      (`"Not authenticated"` vs. `"Could not validate credentials"`),
      confirming it decodes/verifies whatever token is presented before ever
      looking at the request body. A route that requires a valid token to
      reach its own body logic cannot be the first-login step. **Where a
      human/API client obtains their FIRST token is still unconfirmed** — see
      new T0.6.
- [x] T0.2 **(live)** Confirmed 2026-09-07, same session as T0.1: a real,
      stable REST API exists under `/api/v1/`, no direct datastore access
      needed for registration. Scope is org → hv ("hierarchy view") →
      network → device — three levels, not the two this spec originally
      assumed. `epcadopt.Request` updated with the missing `HVID` field;
      `Validate`/`Plan`/CLI (`--hv` flag) updated to match. Exact
      request/response body shapes for the registration calls themselves
      are still unconfirmed — that's Phase 2, not this task.
- [ ] T0.3 **(live)** Confirm which firmware families/versions the target
      controller version actually accepts for adoption — do not carry over an
      unverified version-support claim (see the `fitadopt.MinFitVersion`
      correction elsewhere in this repo's history as the cautionary precedent).
      **Partial progress 2026-09-07:** checked the controller API container's
      own source for a static model/firmware allow-list (grepped for
      supported_model/model_list/allowed_model/firmware_min-style names, then
      inspected the one real hit,
      `SUPPORT_FIRMWARE_OPERATE_PLATFORM_TYPE` in
      `pkg/general/firmware_upgrade` — it's just `{"switch","ap"}`, a
      device-CLASS set, not a model/version allow-list). No compile-time
      allow-list exists; support is almost certainly DB-driven (found the
      app's own Mongo connection target, `epc-db:27017/main`, via its
      `MongoDb`/`MongoUtil` config classes). Did not query it: an
      unauthenticated `pymongo` connection was correctly refused
      (`listCollections requires authentication`), and using the app's own
      already-configured DB credentials to run an ad-hoc query outside its
      normal call path felt like a materially more invasive step than route
      introspection — did not do it unilaterally while a peer session is
      concurrently active against the same DB. Still open; needs either the
      peer's own knowledge (they're driving the live adoption attempt) or a
      deliberate, agreed-upon authenticated read.
- [x] T0.4 **(live)** Confirmed 2026-09-07: device checkin (`POST
      /api/v1/checkin`) uses a distinct, non-JWT scheme — an empty-body probe
      threw a server-side `KeyError: 'id'` inside the controller's own
      checkin-auth handler, consistent with the previously reverse-engineered
      header-based HMAC checkin mechanism documented elsewhere in this
      project's research (not repeated here — protocol mechanics, not secret
      values). Not formally "documented" by the vendor as far as this probe
      could tell; treat as reverse-engineered and version defensively, per
      the original question's framing.
- [x] T0.5 spec.md's "Open questions" §1/§2 updated in place with the
      resolved answers above (2026-09-07), generic — no real IPs/IDs/MACs
      from the instance used to confirm them.
- [ ] T0.6 **(live, blocks T2.1)** Find the real first-login mechanism: where
      does a human operator or the SPA itself obtain its FIRST JWT, given
      `POST /api/v1/jwt-token` is now confirmed to require one already? Candidates
      not yet checked: other routes this session's route-table scan may have
      missed, a different container in the stack (only `epc-api`'s own route
      table was inspected — `epc-raccoon`/`epc-agent`/`epc-otter` were not),
      or an external identity provider the browser SPA calls directly (would
      need a browser network trace of an actual UI login, not container
      introspection).

## Phase 1 — Package skeleton (no live controller needed)
- [x] T1.1 `go/internal/epcadopt/` package created with doc comment
      cross-referencing this spec.
- [x] T1.2 `Request`/`Result`/`Validate` types stubbed (signatures + doc
      comments only), mirroring `fitadopt.Request`/`Result`/`Validate`.
- [x] T1.3 `Expected`/`Observed`/`Prove` types stubbed, mirroring
      `fitadopt.Expected`/`Observed`/`Prove`.
- [x] T1.4 `Client` interface stubbed per plan.md (narrow: Login /
      RegisterDevice / DeviceStatus) — no concrete implementation yet.
- [~] T1.5 `Validate`/`Prove` logic implemented for every precondition that
      does NOT depend on T0.3 (model/scope/real-identity/recovery-route gates,
      full report-every-failure proof). The one remaining piece — gating on
      which firmware families the controller accepts — stays deferred until
      T0.3, and is marked TODO in `eligibility.go` rather than guessed.
- [x] T1.6 `epcadopt.Plan(Request) ([]flash.Step, error)` added (plan.go):
      reuses `flash.Step`, refuses ineligible requests, encodes the FR6
      ordered/gated sequence (backup-before-mutation, pointer-persistence,
      checkin-gate re-assert, real-serial registration, survive-disruption).

## Phase 2 — Controller client implementation
- [ ] T2.1 Concrete `Client` implementation for whichever transport T0.1/T0.2
      confirmed (HTTP API client, or the datastore-seeder escape hatch from
      plan.md, or both if the real mechanism is a hybrid). **Blocked on T0.6**
      for `Login` specifically — do not implement it against the disproven
      `POST /api/v1/jwt-token` grant_type/username/password guess (see
      client.go's doc comment). `RegisterDevice`/`DeviceStatus` are blocked on
      confirming their exact request/response body shapes (unconfirmed, not
      just the URL pattern).
- [ ] T2.2 `httptest`-based unit tests with synthetic fixtures (no real
      captured traffic — regenerate request/response shapes from the
      confirmed protocol, don't paste real payloads that might carry real
      IDs).
- [ ] T2.3 If a datastore-seeder path is needed: an explicit safety comment on
      why it's a distinct, higher-risk interface (per plan.md), plus whatever
      minimal precondition checks are feasible without a live connection.

## Phase 3 — CLI
- [~] T3.1 `go/cmd/pelegrun/epc.go`: `check` subcommand — reads AP identity
      live (reusing crossflash's login+SysInfo path) and reports FR4
      eligibility. Controller reachability/auth is honestly labeled NOT
      VERIFIED rather than faked, because the controller client's real shape
      is still Phase 0 (T0.1/T0.2). Finish when Phase 0 confirms the transport.
- [x] T3.2 `plan` subcommand: builds and prints the ordered, gated plan
      (`epcadopt.Plan`, reusing `flash.Step`), modeled on `cmdOpenWrtPlan`;
      mutates nothing and says so.
- [x] T3.3 `prove` subcommand: modeled on `cmdFitProve` exactly (same
      `--expected`/`--observed` JSON-file pattern).
- [x] T3.4 Wired into `main.go`'s dispatch + usage text, same pattern as the
      `openwrt`/`crossflash` additions.
- [x] T3.5 `runCap`-based CLI tests (`epc_test.go`): usage, missing-flag
      errors, eligible/ineligible check, gated plan + ineligible-plan refusal,
      and prove happy-path + report-every-failure — synthetic `httptest` AP
      only, fake serial/MAC, mirroring `openwrt_test.go`/`crossflash_test.go`.

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
- [x] T5.1 `docs/USAGE.md` "EPC pairing — framework stage" section added,
      matching the "HTTP-only cross-flash" section's style — real command
      sequences (verified against `epc.go`'s actual flags), generic
      placeholder values only, and an honest statement of what `check`/`plan`
      can and can't verify yet.
- [x] T5.2 `README.md`'s command examples block extended with `pelegrun epc
      check|plan|prove`, matching the `crossflash`/`openwrt` entries' style.
- [~] T5.3 `ROADMAP.md` updated with a status row in "Existing work to
      leverage" describing the framework-vs-controller-client split honestly.
      **Not yet promoted to a tracked top-level phase (P13)** — per this
      task's own condition, that's still gated on Phase 0/2 (live controller
      confirmation), which hasn't happened.

## Explicit non-tasks (per spec.md's non-goals)
- Deploying/configuring an EPC controller instance.
- A general-purpose EPC client covering RADIUS/mDNS/VLAN/SSID features.
- Automating controller discovery.
