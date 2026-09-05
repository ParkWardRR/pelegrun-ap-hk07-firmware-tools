# Examples

Runnable fixtures for the fleet (P9) and FIT (P10) command surfaces. They are
exercised by the CLI tests (`go/cmd/pelegrun/*_test.go`), so they stay valid.

Everything below is **read-only** — `fleet plan`, `fit check`, and `fit prove`
change nothing on any device.

## Fleet (P9)

```console
# Build a read-only rollout plan. --now is fixed here so last_seen freshness is
# deterministic; drop it to use the current time.
pelegrun fleet plan \
  --inventory examples/fleet-inventory.json \
  --policy    examples/fleet-policy.json \
  --image ecw230v3-282.bin \
  --image-sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --family cloud --now 2026-09-04T12:05:00Z \
  --out plan.json
```

The sample inventory yields **2 eligible / 1 blocked / 1 canary**: `ap-03-blocked`
is refused because its identity confidence is only `probable` and it has no backup
evidence bundle (the policy requires `verified` + a backup).

```console
# Revalidate the saved plan just before execution (fails closed on any drift).
pelegrun fleet apply \
  --inventory examples/fleet-inventory.json \
  --policy    examples/fleet-policy.json \
  --plan plan.json \
  --image ecw230v3-282.bin \
  --image-sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --family cloud --now 2026-09-04T12:05:00Z --max-age 1h
```

## FIT real-serial adoption (P10)

```console
# P10a: is this device/image/serial combination allowed to adopt?
pelegrun fit check --request examples/fit-request.json

# P10b: after adoption, prove it worked (serial preserved, version/slot/access).
pelegrun fit prove \
  --expected examples/fit-expected.json \
  --observed examples/fit-observed.json
```

Both print human-readable results and exit non-zero if a gate fails.

## Files

| File | Used by |
|---|---|
| `fleet-inventory.json` | `fleet plan` / `fleet apply` — normalized device inventory |
| `fleet-policy.json` | `fleet plan` / `fleet apply` — eligibility + canary + window policy |
| `fit-request.json` | `fit check` — a FIT adoption eligibility request |
| `fit-expected.json` / `fit-observed.json` | `fit prove` — intended vs re-probed state |

> Serials here (`SWLWX420001T`, …) are valid Code27 example values, not real
> device serials. Real FIT adoption uses the device's own serial and never a
> generated one.
