// Command pelegrun — falconry-themed orchestrator to cross-flash and recover
// EnGenius/Senao ap-hk07 (IPQ807x) APs without bricking them.
//
// No arguments launches the TUI; subcommands are script/CI friendly.
// Unofficial; not affiliated with EnGenius or Senao. See README.md / SAFETY.md.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/band"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/jess"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/tui"

	"golang.org/x/term"
)

// Version is stamped via -ldflags "-X main.Version=...".
var Version = "0.4.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point: it dispatches a subcommand, writing normal
// output to out and diagnostics to errw, and returns the process exit code
// (0 ok, 1 command error, 2 unknown command).
func run(args []string, out, errw io.Writer) int {
	cmd := ""
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	var err error
	switch cmd {
	case "", "tui":
		err = runTUI(out)
	case "version":
		fmt.Fprintf(out, "pelegrun %s\n", Version)
	case "plan":
		fmt.Fprint(out, planText)
	case "serial":
		err = cmdSerial(out, args)
	case "snextra":
		err = cmdSnextra(out, args)
	case "check":
		err = cmdCheck(out, args)
	case "envcheck":
		err = cmdEnvcheck(out, args)
	case "discover":
		err = cmdDiscover(out, args)
	case "fleet":
		err = cmdFleet(out, args)
	case "dump":
		err = cmdDump(out, args)
	case "redact":
		err = cmdRedact(out, args)
	case "adapters":
		err = cmdAdapters(out, args)
	case "fit":
		err = cmdFit(out, args)
	case "-h", "--help", "help":
		fmt.Fprint(out, usageText)
	default:
		fmt.Fprintf(errw, "pelegrun: %q is planned but not implemented yet (see ROADMAP.md)\n", cmd)
		return 2
	}
	if err != nil {
		fmt.Fprintf(errw, "error: %v\n", err)
		return 1
	}
	return 0
}

func runTUI(out io.Writer) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprint(out, usageText)
		return nil
	}
	return tui.Run(Version)
}

func argVal(a []string, key string) string {
	for i, v := range a {
		if v == key && i+1 < len(a) {
			return a[i+1]
		}
	}
	return ""
}

func or(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func cmdSerial(out io.Writer, a []string) error {
	model := argVal(a, "--model")
	if model == "" {
		return fmt.Errorf("serial: --model <CODE> required (e.g. X42)")
	}
	s, err := band.MakeSerial(or(argVal(a, "--prefix"), "SWLW"), model, or(argVal(a, "--suffix"), "0001"))
	if err != nil {
		return err
	}
	fmt.Fprintln(out, s)
	return nil
}

func cmdSnextra(out io.Writer, a []string) error {
	model := argVal(a, "--model")
	if model == "" {
		return fmt.Errorf("snextra: --model <CODE> required")
	}
	s, err := band.MakeSnextra(argVal(a, "--prefix"), model)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, s)
	return nil
}

func cmdCheck(out io.Writer, a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("check: <serial> required")
	}
	mc, _ := band.ModelCode(a[0])
	fmt.Fprintf(out, "serial=%s valid=%t model_code=%s\n", a[0], band.ValidateSerial(a[0]), mc)
	if !band.ValidateSerial(a[0]) {
		return fmt.Errorf("check character does not match")
	}
	return nil
}

func cmdEnvcheck(out io.Writer, a []string) error {
	var data []byte
	var err error
	if len(a) > 0 && a[0] != "-" {
		data, err = os.ReadFile(a[0])
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return err
	}
	e := hood.ParsePrintenv(string(data))
	if e.IsComplete() {
		fmt.Fprintln(out, "env: COMPLETE — safe to append individual fields")
		return nil
	}
	fmt.Fprintf(out, "env: INCOMPLETE — missing %v\nrefuse writes; recover with `env default -a` over UART first\n", e.Missing())
	return fmt.Errorf("incomplete env")
}

func cmdDiscover(out io.Writer, a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("discover: <url> required (e.g. http://192.168.1.1)")
	}
	fam, err := eyas.Fingerprint(context.Background(), jess.InsecureClient(10*time.Second), a[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "family=%s\naccess=%s\n", fam, fam.AccessHint())
	return nil
}

const usageText = "pelegrun — cross-flash & recover EnGenius/Senao ap-hk07 APs (unofficial)\n\n" +
	"USAGE:\n" +
	"  pelegrun                 launch the TUI (default)\n" +
	"  pelegrun version | plan\n" +
	"  pelegrun discover <url>            fingerprint firmware family (eyas)\n" +
	"  pelegrun serial  --model X42 [--prefix P --suffix S]   Code27 serial (band)\n" +
	"  pelegrun snextra --model X42 [--prefix P]              20-char field-19 value\n" +
	"  pelegrun check   <serial>                              validate a serial\n" +
	"  pelegrun envcheck [file|-]                             hood env completeness gate\n" +
	"  pelegrun fleet   plan|apply ...                        P9 batch rollout (read-only planner)\n" +
	"  pelegrun dump    plan --dest DIR [--proc-mtd f|-]      on-device full-flash capture plan\n" +
	"  pelegrun redact  [file|-] [--mac] [--value S]...       scrub secrets from a bundle/log\n" +
	"  pelegrun adapters list|validate                        board support registry (P12d)\n" +
	"  pelegrun fit     check|prove ...                       FIT real-serial adoption gates (P10)\n\n" +
	"Image re-head ships as the quarry binary (Rust). Unofficial; hardware you own only.\n"

const planText = "Safety ladder (why UART is usually unnecessary):\n\n" +
	"  1. network flash        no UART — dual A/B slot means a bad image never bricks\n" +
	"  2. network env-repair   no UART — append-only fw_setenv on a verified env\n" +
	"  3. UART env-repair      gated: env default -a -> inspect -> env save\n" +
	"  4. UART TFTP re-flash   truly dead board — lure calls it back over the wire\n\n" +
	"Invariants the tool cannot break: write the INACTIVE slot; env is APPEND-ONLY.\n"
