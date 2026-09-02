// Command swallow — falconry-themed orchestrator to cross-flash and recover
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

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/band"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/jess"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/tui"

	"golang.org/x/term"
)

// Version is stamped via -ldflags "-X main.Version=...".
var Version = "0.3.0"

func main() {
	args := os.Args[1:]
	cmd := ""
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	var err error
	switch cmd {
	case "", "tui":
		err = runTUI()
	case "version":
		fmt.Printf("swallow %s\n", Version)
	case "plan":
		fmt.Print(planText)
	case "serial":
		err = cmdSerial(args)
	case "snextra":
		err = cmdSnextra(args)
	case "check":
		err = cmdCheck(args)
	case "envcheck":
		err = cmdEnvcheck(args)
	case "discover":
		err = cmdDiscover(args)
	case "-h", "--help", "help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "swallow: %q is planned but not implemented yet (see ROADMAP-DEV.md)\n", cmd)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI() error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Print(usageText)
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

func cmdSerial(a []string) error {
	model := argVal(a, "--model")
	if model == "" {
		return fmt.Errorf("serial: --model <CODE> required (e.g. X42)")
	}
	s, err := band.MakeSerial(or(argVal(a, "--prefix"), "SWLW"), model, or(argVal(a, "--suffix"), "0001"))
	if err != nil {
		return err
	}
	fmt.Println(s)
	return nil
}

func cmdSnextra(a []string) error {
	model := argVal(a, "--model")
	if model == "" {
		return fmt.Errorf("snextra: --model <CODE> required")
	}
	s, err := band.MakeSnextra(argVal(a, "--prefix"), model)
	if err != nil {
		return err
	}
	fmt.Println(s)
	return nil
}

func cmdCheck(a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("check: <serial> required")
	}
	mc, _ := band.ModelCode(a[0])
	fmt.Printf("serial=%s valid=%t model_code=%s\n", a[0], band.ValidateSerial(a[0]), mc)
	if !band.ValidateSerial(a[0]) {
		return fmt.Errorf("check character does not match")
	}
	return nil
}

func cmdEnvcheck(a []string) error {
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
		fmt.Println("env: COMPLETE — safe to append individual fields")
		return nil
	}
	fmt.Printf("env: INCOMPLETE — missing %v\nrefuse writes; recover with `env default -a` over UART first\n", e.Missing())
	return fmt.Errorf("incomplete env")
}

func cmdDiscover(a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("discover: <url> required (e.g. http://192.168.1.1)")
	}
	fam, err := eyas.Fingerprint(context.Background(), jess.InsecureClient(10*time.Second), a[0])
	if err != nil {
		return err
	}
	fmt.Printf("family=%s\naccess=%s\n", fam, fam.AccessHint())
	return nil
}

const usageText = "swallow — cross-flash & recover EnGenius/Senao ap-hk07 APs (unofficial)\n\n" +
	"USAGE:\n" +
	"  swallow                 launch the TUI (default)\n" +
	"  swallow version | plan\n" +
	"  swallow discover <url>            fingerprint firmware family (eyas)\n" +
	"  swallow serial  --model X42 [--prefix P --suffix S]   Code27 serial (band)\n" +
	"  swallow snextra --model X42 [--prefix P]              20-char field-19 value\n" +
	"  swallow check   <serial>                              validate a serial\n" +
	"  swallow envcheck [file|-]                             hood env completeness gate\n\n" +
	"Image re-head ships as the quarry binary (Rust). Unofficial; hardware you own only.\n"

const planText = "Safety ladder (why UART is usually unnecessary):\n\n" +
	"  1. network flash        no UART — dual A/B slot means a bad image never bricks\n" +
	"  2. network env-repair   no UART — append-only fw_setenv on a verified env\n" +
	"  3. UART env-repair      gated: env default -a -> inspect -> env save\n" +
	"  4. UART TFTP re-flash   truly dead board — lure calls it back over the wire\n\n" +
	"Invariants the tool cannot break: write the INACTIVE slot; env is APPEND-ONLY.\n"
