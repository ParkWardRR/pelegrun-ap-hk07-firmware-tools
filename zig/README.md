# lure — deep-brick recovery helper (Zig)

The **lure** calls a dead board back over the wire. When even the rootfs is gone
and only the bootloader answers, recovery needs a tiny, dependency-free
**TFTP (RFC 1350) responder** to serve firmware to u-boot's `tftpboot`.

Zig is a good fit: a small, statically-linked responder with no runtime. It talks
to libc sockets directly (`extern "c"` + `link_libc`) because Zig 0.16 removed
`std.net` and the `std.posix` socket wrappers — the libc surface is the stable,
portable one.

## Status: implemented + tested

- `zig build` compiles the `lure` binary (Zig **0.16**).
- `zig build test` runs the unit tests for the parsing helpers (`be16`,
  `parseRRQ`).
- `./tftp_test.sh` is an integration test: it serves a temp dir and verifies a
  real multi-block transfer round-trips byte-for-byte, plus a clean
  file-not-found path and that the server survives an error.
- All three run in local CI (`make ci` from the repo root) and via `make test`.

## Usage

```sh
zig build                 # -> zig-out/bin/lure
cd /path/with/firmware && /path/to/lure    # serves the current dir over TFTP (:6969)
```

Recovery normally wants port `:69` (which needs root); the default `:6969` keeps
the tests unprivileged. See [`../docs/USAGE.md`](../docs/USAGE.md) for the full
UART / `tftpboot` recovery flow and [`../SAFETY.md`](../SAFETY.md) for the
no-brick invariants.
