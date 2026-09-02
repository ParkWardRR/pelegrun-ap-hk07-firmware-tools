// Command swallow is the falconry-themed orchestrator for cross-flashing and
// recovering EnGenius/Senao ap-hk07 (IPQ807x) access points without UART where
// possible, and with a gated UART path where it isn't.
//
// Unofficial; not affiliated with EnGenius or Senao. For interoperability and
// self-hosting on hardware you own. See README.md and SAFETY.md.
//
// This is the dev-phase-1 skeleton: it prints its plan and hands the working
// image/serial math to the quarry binary (Rust). Network/UART flows land in
// later phases behind the internal falconry packages.
package main

import (
	"flag"
	"fmt"
	"os"
)

// Version is stamped at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0"

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		return
	}
	switch args[0] {
	case "version":
		fmt.Printf("swallow %s\n", Version)
	case "plan":
		fmt.Print(plan)
	default:
		fmt.Fprintf(os.Stderr, "swallow: %q is planned but not implemented yet (see ROADMAP-DEV.md)\n", args[0])
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(usageText)
}

const usageText = "swallow — cross-flash & recover EnGenius/Senao ap-hk07 APs (unofficial)\n\n" +
	"USAGE:\n" +
	"  swallow version\n" +
	"  swallow plan            print the safety ladder + what each phase adds\n\n" +
	"PLANNED (see ROADMAP-DEV.md):\n" +
	"  swallow discover        eyas:    find + fingerprint an AP\n" +
	"  swallow backup          mews:    dump mtd7/8/11 + config, hashed\n" +
	"  swallow serial          band:    provision a unique Code27 / snextra (quarry)\n" +
	"  swallow rehead          quarry:  one-field product_id patch (available now via quarry)\n" +
	"  swallow flash           hood+jess: safe A/B flash, env-completeness gated\n" +
	"  swallow recover         creance+lure: gated UART env repair / TFTP re-flash\n\n" +
	"Today the image + serial math ships as the tested quarry binary (Rust);\n" +
	"swallow grows the safe network/UART automation around it, phase by phase.\n"

const plan = "Safety ladder (why UART is usually unnecessary):\n\n" +
	"  1. network flash        no UART — dual A/B slot means a bad image never bricks\n" +
	"  2. network env-repair   no UART — append-only fw_setenv on a verified env\n" +
	"  3. UART env-repair      gated: env default -a -> inspect -> env save\n" +
	"  4. UART TFTP re-flash   truly dead board — lure calls it back over the wire\n\n" +
	"Two invariants the tool structurally cannot break:\n" +
	"  - always write the INACTIVE slot; keep the active one bootable\n" +
	"  - the env is APPEND-ONLY; never erase or hand-rebuild it (the one thing that bricks)\n"
