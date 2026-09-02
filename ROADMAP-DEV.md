# Development roadmap (6 phases)

Engineering execution order. Detailed task list: [specs/001-swallow-mvp/tasks.md](specs/001-swallow-mvp/tasks.md).

```mermaid
flowchart LR
  P1[Phase 1 ✅<br/>Core + scaffold] --> P2[Phase 2<br/>Access + discovery]
  P2 --> P3[Phase 3<br/>Safety + backup + provision]
  P3 --> P4[Phase 4<br/>No-UART A/B flash]
  P4 --> P5[Phase 5<br/>UART recovery]
  P5 --> P6[Phase 6<br/>Release polish]
  P4 --> P5
  style P1 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P2 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P3 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P4 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P5 fill:#1f7a1f,stroke:#0a3,color:#fff
```

| Phase | Scope | Deliverable | Status |
|-------|-------|-------------|--------|
| 1 | Core + scaffold | tested `quarry` (Rust) + `swallow`/Zig skeletons + Spec Kit + CI | ✅ done |
| 2 | Access + discovery | `eyas` fingerprint, `jess` ssh/cloud/luci adapters | ✅ done |
| 3 | Safety + backup + provision | `hood` env gate, `mews` bundles, `band` serials | ✅ done |
| 4 | No-UART A/B flash | slot-aware flash + verify + rollback | ✅ done |
| **5** | **UART recovery** | `creance` gated serial repair, `lure` Zig TFTP responder | **✅ done (this release)** |
| 6 | Release polish | cross-compiled binaries, fixture/property tests, docs | ⬜ next |

**Phases 1–5 complete.** Rust 12/12, Go 6 packages, Zig build + `lure` TFTP
integration test, and termwright terminal-E2E all green. Phase 6 is release
polish: cross-compiled binaries, recorded-fixture tests, and expanded docs.
