# Using swallow

A practical walkthrough: install, the guided TUI, the individual commands, and
how the safe-flash and recovery flows fit together.

> **Unofficial — not affiliated with EnGenius or Senao.** For interoperability
> and self-hosting on hardware you own. Cross-flashing can brick hardware; read
> [`../SAFETY.md`](../SAFETY.md) first.

## Install

### Prebuilt binaries (recommended)

Grab the archive for your platform from the
[latest release](https://github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/releases/latest),
then **verify the checksum** before running:

```sh
# example: Apple Silicon macOS
curl -LO https://github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/releases/latest/download/swallow-darwin-arm64
curl -LO https://github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/releases/latest/download/SHA256SUMS
shasum -a 256 -c SHA256SUMS --ignore-missing   # or: sha256sum -c
chmod +x swallow-darwin-arm64
./swallow-darwin-arm64
```

Binaries are published for `darwin/{arm64,amd64}`, `linux/{amd64,arm64}`, and
`windows/amd64` (the `lure` recovery helper is POSIX-only: macOS + Linux).

### From source

```sh
git clone https://github.com/ParkWardRR/swallow-ap-hk07-firmware-tools
cd swallow-ap-hk07-firmware-tools
cd go && go build -o swallow ./cmd/swallow    # the CLI/TUI (Go)
cargo build -p quarry --release               # image re-head + serial core (Rust)
cd zig && zig build                           # lure TFTP recovery helper (Zig)
```

## The guided TUI

Run `swallow` with no arguments to open the dashboard. The sidebar walks the job
in plain steps; each screen shows **live output from the real logic** — including
the safety refusals — not mock data.

| Step | What it does |
|------|--------------|
| **Discover** | Fingerprints the firmware family (Cloud · EWS/LuCI · FIT) from its web UI |
| **Connect** | Shows how to reach each family — SSH is on **:8822**, not 22 |
| **Back Up** | The required read-only evidence bundle to capture *before* flashing |
| **Safeguards** | Why the tool can't brick: the env gate refuses wiped/empty writes |
| **Identity** | Mints a unique, collision-checked serial for the target model |
| **Install** | The no-UART A/B flash: write the spare slot, reboot, re-verify |
| **Verify** | Confirms the device came back as intended; rollback if not |

Navigate with `↑ ↓` (or `j k`), jump with `g` / `G`, quit with `q`.

## Commands

Everything the TUI shows is also scriptable:

```sh
swallow discover http://192.168.1.1     # → family + which access adapter to use
swallow serial  --model X42 --prefix SWLW --suffix 0001   # → SWLWX420001T
swallow snextra --model X42             # → 20-char u-boot field-19 value
swallow check   EPC1X4200011            # → serial=… valid=true model_code=X42
swallow envcheck env.txt                # completeness gate; refuses if incomplete
swallow plan                            # print the ordered, gated flash plan
swallow version
```

`swallow envcheck` reads a `fw_printenv` dump (file, or `-` for stdin) and exits
non-zero if the bootloader env is incomplete — the one state that bricks:

```console
$ printf 'ethaddr=00:03:7f:12:3e:87\n' | swallow envcheck -
env: INCOMPLETE — missing [active_fw app_part bootcmd rootfsname]
refuse writes; recover with `env default -a` over UART first
```

Model codes (the 3 chars at serial positions 5–7): `X42` ECW230v3 · `X44`
EWS377AP v3 · `X45` EWS377-FIT. See the field guide's
[model-codes](https://github.com/ParkWardRR/engenius-field-guide/blob/main/model-codes.md)
for the full table.

### Image re-head (quarry)

The one-field `product_id` patch that lets a sibling's image pass another model's
upload gate ships as the standalone `quarry` binary:

```sh
quarry inspect firmware.bin             # parse + validate the Senao header
quarry rehead firmware.bin --to 284     # ECW230v3; writes only 4 bytes at 0x08
```

## The two invariants (why it won't brick)

1. **Writes go to the `INACTIVE` A/B slot.** The running slot stays bootable — a
   bad image is undone with a factory-reset hold.
2. **The bootloader env is append-only.** The tool only adds fields to an env it
   has verified complete; it can't erase it or save a partial one. A
   valid-but-incomplete env is the single thing that bricks, and the code is
   structurally incapable of producing one.

## If it won't boot (UART recovery)

Rarely needed, but if a board is truly dead you'll want a USB-TTL adapter on the
console header. Two helpers cover it:

- **creance** drives the gated *"prove before persist"* u-boot env repair:
  `printenv` → `env default -a` (RAM only) → inspect → `env save` → `env load` →
  verify → **cold boot** (full power removal, not just `reset`).
- **lure** is a tiny TFTP responder for `tftpboot` recovery — point the board's
  u-boot at your machine and serve it a known-good image:

  ```sh
  cd /path/with/firmware && lure       # serves the current dir over TFTP
  ```

See [`../SAFETY.md`](../SAFETY.md) and the field guide's
[cross-flash walkthrough](https://github.com/ParkWardRR/engenius-field-guide/blob/main/crossflash-ews377apv3-walkthrough.md)
for the full hardware procedure and evidence checklist.
