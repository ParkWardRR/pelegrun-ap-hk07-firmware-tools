# lure — deep-brick recovery helper (Zig) · PLANNED (dev phase 5)

The **lure** calls a dead board back over the wire. When even the rootfs is gone
and only the bootloader answers, `swallow recover` needs a tiny, dependency-free
**TFTP / BOOTP responder** to serve firmware to u-boot's `tftpboot`.

Zig is a good fit: a freestanding, statically-linked responder that ships inside
the `swallow` binary with no runtime. This is a stub — implementation lands in
dev phase 5 (see `../ROADMAP-DEV.md`). Not yet built in CI.
