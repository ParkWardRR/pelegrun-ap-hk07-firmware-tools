package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/dump"
)

// cmdDump plans an on-device, verified full-flash capture — the safety net you
// take before any flash. It is read-only and plan-only here: it prints the exact
// device-local dump script (and optionally a manifest) for an accessor to run.
// The device's own /proc/mtd drives the layout; never hard-code mtd indices.
func cmdDump(out io.Writer, a []string) error {
	if len(a) == 0 || a[0] != "plan" {
		fmt.Fprint(out, dumpUsage)
		if len(a) == 0 {
			return nil
		}
		return fmt.Errorf("dump: unknown subcommand %q (only `plan` for now)", a[0])
	}
	a = a[1:]

	dest := argVal(a, "--dest")
	if dest == "" {
		return fmt.Errorf("dump plan: --dest <device-local dir> required (e.g. /tmp/swallow-dump)")
	}
	if err := dump.ValidateDest(dest); err != nil {
		return err
	}

	// Read /proc/mtd from a file, or stdin when omitted or "-".
	src := argVal(a, "--proc-mtd")
	var data []byte
	var err error
	if src == "" || src == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(src)
	}
	if err != nil {
		return fmt.Errorf("read /proc/mtd: %w", err)
	}

	parts, err := dump.ParseProcMTD(string(data))
	if err != nil {
		return err
	}

	opts := dump.PlanOptions{
		NAND:   has(a, "--nand"),
		Model:  argVal(a, "--model"),
		Serial: argVal(a, "--serial"),
	}
	steps, err := dump.Plan(parts, dest, opts)
	if err != nil {
		return err
	}

	if ok, note := dump.SafeDestHint(dest); !ok {
		fmt.Fprintf(out, "WARNING: %s\n", note)
	}
	fmt.Fprintf(out, "on-device dump plan → %s  (%d partitions, ~%s, %s)\n",
		dest, len(parts), humanBytes(dump.EstimatedBytes(parts)), dumpTool(opts))
	for _, s := range steps {
		gate := " "
		if s.Gate {
			gate = "!"
		}
		fmt.Fprintf(out, "  %s %s\n      $ %s\n", gate, s.Desc, s.Command)
	}

	if mfPath := argVal(a, "--manifest"); mfPath != "" {
		m := dump.NewManifest(parts, dest, "swallow "+Version, opts)
		if err := m.Save(mfPath); err != nil {
			return fmt.Errorf("write manifest: %w", err)
		}
		fmt.Fprintf(out, "manifest (planned) written to %s\n", mfPath)
	}
	fmt.Fprintln(out, "\n(read-only plan; nothing was run. Verify hashes after capture, then pull to host.)")
	return nil
}

func dumpTool(o dump.PlanOptions) string {
	if o.NAND {
		return "nanddump"
	}
	return "dd"
}

func has(a []string, flag string) bool {
	for _, v := range a {
		if v == flag {
			return true
		}
	}
	return false
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

const dumpUsage = "swallow dump — plan an on-device, verified full-flash capture (safety net)\n\n" +
	"USAGE:\n" +
	"  swallow dump plan --dest /tmp/swallow-dump [--proc-mtd file|-] [--nand] \\\n" +
	"        [--model ap-hk07] [--serial <s>] [--manifest out.json]\n\n" +
	"Reads the device's /proc/mtd (stdin by default), then prints the read-only\n" +
	"dd/nanddump script that writes every partition to --dest ON THE DEVICE and hashes\n" +
	"it. Store --dest on tmpfs/USB (NOT the NAND being dumped). Nothing is run here.\n"
