# OpenWrt on the EnGenius EWS377AP v3 — install & restore

Community/unofficial **OpenWrt** port for the EnGenius **EWS377AP v3** (`ap-hk07`,
Qualcomm IPQ8072A). Built on the NSS-EDMA OpenWrt fork (kernel 6.18, NSS offload).
Validated end-to-end on hardware: persistent NAND boot, Ethernet, WiFi (WPA2), and
config surviving reboots.

> **Downloads:** get the images + `SHA256SUMS` from this repo's
> [Releases](https://github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/releases)
> (tag `openwrt-ews377ap-v3-v0.1`). Source: fork branch `ews377ap-v3` of
> `openwrt-nss-edma`.

## ⚠️ Read this first
- **You can brick your AP.** This is unofficial and overwrites the OEM firmware.
- **Only the UART + u-boot method (Method A) is hardware-proven.** The web-upload
  `.bin` (Method B) matches the OEM format but is **community-untested** — only try
  it with UART recovery on hand.
- **Single-slot install:** OpenWrt lands on slot 0 (the `rootfs` partition); there
  is **no on-device OEM fallback** afterward. **Back up your own NAND first**
  (Part 1) — your MAC and radio calibration are unique to your unit; never reuse
  someone else's dump.
- **Never erase or write the ART partition or the bootloader region** (`0x0`–
  `0x1000000`). ART holds your calibration + MAC; losing it is a true brick.
- Secure boot was **unfused** on the test unit (custom images boot). If yours is
  fused, custom images won't boot and you'd need to restore OEM.
- Always **verify `SHA256SUMS`** before flashing.

## Images
| File | Purpose | Status |
|---|---|---|
| `...-squashfs-factory.ubi` | bare UBI — Method A (u-boot `nand write` to slot 0) | ✅ proven |
| `...-initramfs-uImage.itb` | RAM boot via u-boot (dry-run / recovery, nothing written) | ✅ proven |
| `...-squashfs-sysupgrade.bin` | upgrades once already on OpenWrt (`sysupgrade`) | standard |
| `...-web-ui-factory.bin` | Senao image for the OEM web/LuCI updater (Method B) | ⚠️ untested |
| `...-squashfs-qsdk-factory.itb` | QSDK FIT for the OEM CLI updater (Method C) | ⚠️ untested |

## Partition map (256 MiB NAND)
| Region | Offset | Size | Note |
|---|---|---|---|
| bootloader / config / **ART** | `0x0`–`0x1000000` | 16 MiB | **never touch** |
| `rootfs` (slot 0) | `0x1000000` | 111 MiB | ← OpenWrt goes here |
| `0:wififw` | `0x7f00000` | 9 MiB | leave as-is |
| `rootfs_1` (slot 1) | `0x8800000` | 111 MiB | OEM A/B slot (unused by OpenWrt) |
| `0:wififw_1` | `0xf700000` | 9 MiB | leave as-is |

Slot is chosen by the u-boot `active_fw` variable (`0` = slot 0).

## Prerequisites
- USB-TTL serial adapter (**3.3V**) on the console header **J2**, **115200 8N1**.
- A TFTP server on your PC, reachable from the AP over LAN.
- The downloaded images, checksums verified.

---

## Part 1 — Back up YOUR device first (do not skip)
Interrupt u-boot at the boot prompt, then dump every OS/critical partition to TFTP
and keep the files somewhere safe. Adjust offsets/sizes from the map above:
```
setenv ipaddr <ap-ip> ; setenv serverip <tftp-ip> ; ping $serverip
nand read  0x44000000 0x1000000 0x6f00000 ; tftpput 0x44000000 0x6f00000 oem-rootfs.bin
nand read  0x44000000 0x8800000 0x6f00000 ; tftpput 0x44000000 0x6f00000 oem-rootfs_1.bin
nand read  0x44000000 0x7f00000 0x900000  ; tftpput 0x44000000 0x900000  oem-wififw.bin
nand read  0x44000000 0xf700000 0x900000  ; tftpput 0x44000000 0x900000  oem-wififw_1.bin
```
Record your `printenv` output too (especially `ethaddr`, `sn`, and any serial
fields). These backups are your only rollback once you overwrite slot 0.

## Part 2 — Install OpenWrt (Method A, proven)
Optional dry run first — boot OpenWrt entirely in RAM (nothing written), look
around, then power-cycle back to stock:
```
tftpboot 0x44000000 openwrt-...-initramfs-uImage.itb
bootm 0x44000000
```
Flash to slot 0:
```
setenv ipaddr <ap-ip> ; setenv serverip <tftp-ip> ; ping $serverip
tftpboot 0x44000000 openwrt-...-squashfs-factory.ubi
#   $filesize must read 0xe80000 (≈15.2 MB); if not, STOP
nand device 0
nand erase 0x1000000 0x6f00000
nand write 0x44000000 0x1000000 0xe80000
setenv active_fw 0 ; saveenv
reset
```
`nand write` automatically skips the one factory-marked bad block; UBI tolerates
bad blocks. The AP should boot to `root@OpenWrt:~#`; LuCI/LAN is `192.168.1.1`.

## Part 3 — OEM web updater (Method B, experimental / untested)
Upload `...-web-ui-factory.bin` through the stock EnGenius web UI / LuCI firmware
updater. It carries a flash script that erases slot 0 and writes OpenWrt. **If it's
rejected or the flash is interrupted, you may need Method A to recover** — only try
this with UART on hand, and please report the result.

## Part 4 — First boot & upgrades
- Set a root password, configure WiFi (use WPA2/WPA3 — don't leave an open SSID).
- Later upgrades stay on OpenWrt: `sysupgrade -n openwrt-...-squashfs-sysupgrade.bin`
  (or via LuCI → System → Backup/Flash).

---

## Part 5 — Restore back to stock EnGenius firmware
You have two routes:

**Route 1 — from your Part 1 backup (most reliable).** In u-boot, write your OEM
`rootfs` dump back to slot 0:
```
setenv ipaddr <ap-ip> ; setenv serverip <tftp-ip> ; ping $serverip
tftpboot 0x44000000 oem-rootfs.bin
nand device 0
nand erase 0x1000000 0x6f00000
nand write 0x44000000 0x1000000 <size-of-oem-rootfs.bin-in-hex>
setenv active_fw 0 ; saveenv
reset
```
If you also overwrote the wififw partition, restore `oem-wififw.bin` to `0x7f00000`
the same way. Restore your `ethaddr`/serial env fields if they changed.

**Route 2 — official EnGenius `.bin` via the OEM updater.** Once stock boots (via
Route 1), you can re-apply any official EnGenius EWS377AP v3 firmware through the
normal web/cloud updater.

**Never write the ART partition or the region below `0x1000000`.** Because secure
boot is unfused, u-boot + TFTP can always recover the OS as long as ART and the
bootloader are intact.

---

## Why it works (for the curious)
Two device-specific details were required to boot OpenWrt from NAND on this board:
1. The FIT kernel image must expose a config named **`config@hk07`** — the OEM
   u-boot `bootipq` selects the FIT config by board name and aborts on anything
   else.
2. OpenWrt must live on **slot 0** (`rootfs` @`0x1000000`) — its root-mount always
   targets the partition labeled `rootfs`, regardless of which slot the bootloader
   loaded the kernel from.

Full engineering write-up: `PORT-STATUS-ews377ap-v3.md` on the `ews377ap-v3` branch
of the `openwrt-nss-edma` fork.
