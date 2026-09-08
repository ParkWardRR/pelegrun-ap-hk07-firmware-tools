package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/jess"
)

// cmdCrossflash drives the cloud-firmware HTTP upload/upgrade mechanism
// (jess.Cloud) for cross-flashing between ap-hk07 firmware families —
// EWS377AP v3 / EWS377-FIT / ECW230v3 — without UART. `check` is read-only
// (login + identity); `push` stages an already-re-headed image and, only
// with --yes, triggers the flash.
//
// The key fact this command exists to operationalize: upload.cgi validates
// the image's Senao header product_id against the RUNNING firmware's own
// identity, not the target family. Re-head with `quarry rehead <img> <img>
// --to <id>` to the id `check` reports BEFORE calling `push` — pushing an
// image headed for the target family will be rejected.
//
// This path writes the device's *inactive* A/B slot via the OEM's own
// updater, same as a same-family firmware update. It is not UART, and it is
// not guaranteed to boot on every image: a marginal NAND block in the spare
// slot has been observed (on at least one unit) to fail for some community
// UBI layouts while succeeding for every genuine EnGenius image tried
// (FIT and ECW230v3) — keep UART available as the recovery route regardless.
func cmdCrossflash(out io.Writer, a []string) error {
	sub := ""
	if len(a) > 0 {
		sub = a[0]
		a = a[1:]
	}
	switch sub {
	case "check":
		return cmdCrossflashCheck(out, a)
	case "push":
		return cmdCrossflashPush(out, a)
	case "", "-h", "--help", "help":
		fmt.Fprint(out, crossflashUsage)
		return nil
	default:
		return fmt.Errorf("crossflash: unknown subcommand %q (check|push)", sub)
	}
}

func crossflashClient(a []string) (*jess.Cloud, string, error) {
	ap := argVal(a, "--ap")
	if ap == "" {
		return nil, "", fmt.Errorf("--ap <ip-or-host> required")
	}
	user := or(argVal(a, "--user"), "admin")
	pass := or(argVal(a, "--pass"), "admin")
	c := jess.NewCloud("https://" + ap)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Login(ctx, user, pass); err != nil {
		return nil, "", fmt.Errorf("login %s: %w", ap, err)
	}
	return c, ap, nil
}

func cmdCrossflashCheck(out io.Writer, a []string) error {
	c, ap, err := crossflashClient(a)
	if err != nil {
		return err
	}
	info, err := c.SysInfo(context.Background())
	if err != nil {
		return fmt.Errorf("sys_info %s: %w", ap, err)
	}
	fmt.Fprintf(out, "reachable — %s\n", ap)
	fmt.Fprintf(out, "  firmware : %s\n", info.Firmware)
	fmt.Fprintf(out, "  serial   : %s\n", info.Serial)
	fmt.Fprintf(out, "  mac      : %s\n", info.MAC)
	fmt.Fprint(out, "\nre-head any image you push to match ONE of these product_ids\n"+
		"(pick by the family this AP's upload gate wants, i.e. this AP's OWN\n"+
		"identity above — not the family you're pushing TO):\n"+
		"  282  EWS377AP v3\n"+
		"  300  EWS377-FIT\n"+
		"  284  ECW230v3\n"+
		"e.g. quarry rehead image.bin image-reheaded.bin --to 284\n")
	return nil
}

func cmdCrossflashPush(out io.Writer, a []string) error {
	c, ap, err := crossflashClient(a)
	if err != nil {
		return err
	}
	imgPath := argVal(a, "--image")
	if imgPath == "" {
		return fmt.Errorf("crossflash push: --image <re-headed.bin> required")
	}
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return fmt.Errorf("crossflash push --image: %w", err)
	}

	ctx := context.Background()
	size, checksum, err := c.UploadImage(ctx, imgPath, data)
	if err != nil {
		return fmt.Errorf("upload to %s: %w", ap, err)
	}
	fmt.Fprintf(out, "staged on %s: %d bytes, checksum %s\n", ap, size, checksum)
	if size != len(data) {
		return fmt.Errorf("staged size %d does not match local file size %d — refusing to flash", size, len(data))
	}

	if !hasFlag(a, "--yes") {
		fmt.Fprintln(out, "image staged and verified. Re-run with --yes to flash it now (destructive:\n"+
			"writes the device's inactive A/B slot; no rollback slot exists after this\n"+
			"for firmware families whose kernel hardcodes a fixed root partition).")
		return nil
	}

	if err := c.FwUpgrade(ctx); err != nil {
		return fmt.Errorf("fw_upgrade %s: %w", ap, err)
	}
	fmt.Fprintf(out, "fw_upgrade triggered on %s — device is flashing and will reboot; watch UART/console\n", ap)
	return nil
}

func hasFlag(a []string, key string) bool {
	for _, v := range a {
		if v == key {
			return true
		}
	}
	return false
}

const crossflashUsage = "pelegrun crossflash — HTTP-only firmware cross-flash for ap-hk07 (no UART)\n\n" +
	"USAGE:\n" +
	"  pelegrun crossflash check --ap <ip> [--user admin] [--pass admin]\n" +
	"      login + identity: which firmware/product_id is currently running,\n" +
	"      and which product_id to re-head your target image to.\n" +
	"  pelegrun crossflash push --ap <ip> --image <re-headed.bin> [--yes]\n" +
	"      stage + validate (always); flash only with --yes (destructive gate).\n\n" +
	"Re-head the image to match the RUNNING firmware's product_id first:\n" +
	"  quarry rehead image.bin image-reheaded.bin --to <282|300|284>\n" +
	"(the upload gate checks against what's running now, not your target family —\n" +
	"run `crossflash check` to see which id that is). Writes the inactive A/B\n" +
	"slot via the device's own updater; keep UART available as the recovery route.\n"
