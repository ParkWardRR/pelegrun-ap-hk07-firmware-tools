# Development roadmap (6 phases)

Engineering execution order. Detailed task list: [specs/001-swallow-mvp/tasks.md](specs/001-swallow-mvp/tasks.md).

```mermaid
flowchart LR
  P1[Phase 1 ✅<br/>Core + scaffold] --> P2[Phase 2<br/>Access + discovery]
  P2 --> P3[Phase 3<br/>Safety + backup + provision]
  P3 --> P4[Phase 4<br/>No-UART A/B flash]
  P4 --> P5[Phase 5<br/>UART recovery]
  P5 --> P6[Phase 6<br/>Release polish]
  style P1 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P2 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P3 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P4 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P5 fill:#1f7a1f,stroke:#0a3,color:#fff
  style P6 fill:#1f7a1f,stroke:#0a3,color:#fff
```

| Phase | Scope | Deliverable | Status |
|-------|-------|-------------|--------|
| 1 | Core + scaffold | tested `quarry` (Rust) + `swallow`/Zig skeletons + Spec Kit + CI | ✅ done |
| 2 | Access + discovery | `eyas` fingerprint, `jess` ssh/cloud/luci adapters | ✅ done |
| 3 | Safety + backup + provision | `hood` env gate, `mews` bundles, `band` serials | ✅ done |
| 4 | No-UART A/B flash | slot-aware flash + verify + rollback | ✅ done |
| 5 | UART recovery | `creance` gated serial repair, `lure` Zig TFTP responder | ✅ done |
| **6** | **Release polish** | cross-compiled binaries + checksums, fixture/property tests, user docs | **✅ done (this release)** |

**All 6 phases complete.** Rust unit + property tests (5 invariants), Go 7
packages incl. recorded fixtures, Zig `lure` TFTP integration test, and
termwright terminal-E2E all green. Releases ship cross-compiled binaries
(Go ×5, Zig `lure` ×4) with `SHA256SUMS` via a tag-triggered workflow.
