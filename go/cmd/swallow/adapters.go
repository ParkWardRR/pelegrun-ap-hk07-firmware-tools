package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/adapter"
)

// cmdAdapters inspects the board support registry (P12d model DB). `list` shows
// each adapter's tier, capabilities, and whether it may flash; `validate` checks
// every record for internal consistency and exits non-zero if any is broken.
// With no --file it uses the builtin default registry (ap-hk07 = experimental).
func cmdAdapters(out io.Writer, a []string) error {
	sub := ""
	if len(a) > 0 {
		sub = a[0]
		a = a[1:]
	}
	reg, err := adapter.LoadRegistry(argVal(a, "--file"))
	if err != nil {
		return err
	}

	switch sub {
	case "", "list":
		listAdapters(out, reg)
		return nil
	case "validate":
		probs := reg.Validate()
		if len(probs) == 0 {
			fmt.Fprintf(out, "registry OK — %d adapter(s), all consistent\n", len(reg.Adapters))
			return nil
		}
		models := make([]string, 0, len(probs))
		for m := range probs {
			models = append(models, m)
		}
		sort.Strings(models)
		for _, m := range models {
			for _, e := range probs[m] {
				fmt.Fprintf(out, "%s: %v\n", m, e)
			}
		}
		return fmt.Errorf("%d adapter(s) failed validation", len(probs))
	case "-h", "--help", "help":
		fmt.Fprint(out, adaptersUsage)
		return nil
	default:
		return fmt.Errorf("adapters: unknown subcommand %q (list|validate)", sub)
	}
}

func listAdapters(out io.Writer, reg *adapter.Registry) {
	adapters := append([]adapter.Support(nil), reg.Adapters...)
	sort.Slice(adapters, func(i, j int) bool { return adapters[i].Model < adapters[j].Model })
	for _, s := range adapters {
		flash := "flash:no"
		if err := s.CanFlash(); err == nil {
			flash = "flash:YES"
		}
		caps := s.CapabilityList()
		names := make([]string, len(caps))
		for i, c := range caps {
			names[i] = string(c)
		}
		fmt.Fprintf(out, "%-14s tier=%-12s %s\n", s.Model, s.Tier, flash)
		fmt.Fprintf(out, "    caps: %v\n", names)
	}
}

const adaptersUsage = "swallow adapters — inspect the board support registry (P12d)\n\n" +
	"USAGE:\n" +
	"  swallow adapters list     [--file registry.json]   tier + capabilities + flashability\n" +
	"  swallow adapters validate [--file registry.json]   check every record's consistency\n\n" +
	"With no --file, the builtin default registry is used (ap-hk07 = experimental).\n"
