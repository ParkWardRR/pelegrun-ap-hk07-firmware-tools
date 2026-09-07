package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/adapter"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/flash"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/hood"
)

// openWrtModel is the registry key for the OpenWrt flash target (distinct
// from "ap-hk07", the FIT target on the same physical board — see
// adapter.DefaultRegistry's doc comment for why).
const openWrtModel = "ap-hk07-openwrt"

// cmdOpenWrt exposes the mainline/community OpenWrt install path: `check` is
// the read-only registry-eligibility gate (is this target even flash-ready
// per the adapter contract?), `plan` builds and prints the ordered,
// fixed-partition flash plan (flash.PlanOpenWrt). Neither changes anything on
// any device — the actual `nand erase`/`nand write` sequence happens over
// UART, by hand or via a future accessor; see
// openwrt-ews377ap-v3/results-2026-09-06/STAGE3-SLOT0-INSTALL-RESULTS.md for
// the real-hardware procedure this plan encodes.
//
// Unlike `fit`, this command cross-checks the adapter registry before
// building a plan — refusing up front if the target isn't declared
// flash-ready, rather than only encoding the write-order invariants.
func cmdOpenWrt(out io.Writer, a []string) error {
	sub := ""
	if len(a) > 0 {
		sub = a[0]
		a = a[1:]
	}
	switch sub {
	case "check":
		return cmdOpenWrtCheck(out, a)
	case "plan":
		return cmdOpenWrtPlan(out, a)
	case "", "-h", "--help", "help":
		fmt.Fprint(out, openWrtUsage)
		return nil
	default:
		return fmt.Errorf("openwrt: unknown subcommand %q (check|plan)", sub)
	}
}

func cmdOpenWrtCheck(out io.Writer, a []string) error {
	reg, err := adapter.LoadRegistry(argVal(a, "--file"))
	if err != nil {
		return err
	}
	s := reg.Get(openWrtModel)
	if s == nil {
		return fmt.Errorf("openwrt check: %q is not in the registry", openWrtModel)
	}
	if errs := s.Validate(); len(errs) > 0 {
		fmt.Fprintf(out, "%s: NOT CONSISTENT:\n", openWrtModel)
		for _, e := range errs {
			fmt.Fprintf(out, "  - %v\n", e)
		}
		return fmt.Errorf("%d contract problem(s)", len(errs))
	}
	if err := s.CanFlash(); err != nil {
		fmt.Fprintf(out, "%s: NOT FLASH-READY: %v\n", openWrtModel, err)
		return err
	}
	fmt.Fprintf(out, "ELIGIBLE — %s (tier=%s) is flash-ready: %v\n", openWrtModel, s.Tier, s.CapabilityList())
	if s.Has(adapter.CapFlashAB) {
		fmt.Fprintln(out, "note: declares flash_ab — unexpected for this target, double-check the registry")
	}
	return nil
}

func cmdOpenWrtPlan(out io.Writer, a []string) error {
	reg, err := adapter.LoadRegistry(argVal(a, "--file"))
	if err != nil {
		return err
	}
	s := reg.Get(openWrtModel)
	if s == nil || s.CanFlash() != nil {
		reason := "not in the registry"
		if s != nil {
			reason = s.CanFlash().Error()
		}
		return fmt.Errorf("openwrt plan: %s is not flash-ready (%s) — run `pelegrun openwrt check` first", openWrtModel, reason)
	}

	envPath := argVal(a, "--env")
	if envPath == "" {
		return fmt.Errorf("openwrt plan: --env <printenv.txt> required")
	}
	image := argVal(a, "--image")
	if image == "" {
		return fmt.Errorf("openwrt plan: --image <factory.ubi> required")
	}
	envData, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("openwrt plan --env: %w", err)
	}
	env := hood.ParsePrintenv(string(envData))

	steps, err := flash.PlanOpenWrt(env, image)
	if err != nil {
		return fmt.Errorf("openwrt plan: %w", err)
	}
	for i, st := range steps {
		gate := ""
		if st.Gate {
			gate = " [GATE]"
		}
		fmt.Fprintf(out, "%d. %s%s\n    %s\n", i+1, st.Desc, gate, st.Note)
	}
	return nil
}

const openWrtUsage = "pelegrun openwrt — mainline/community OpenWrt install (read-only planning)\n\n" +
	"USAGE:\n" +
	"  pelegrun openwrt check [--file registry.json]\n" +
	"      registry eligibility: is ap-hk07-openwrt declared flash-ready?\n" +
	"  pelegrun openwrt plan --env printenv.txt --image factory.ubi [--file registry.json]\n" +
	"      ordered, gated flash plan (flash.PlanOpenWrt) — fixed rootfs partition,\n" +
	"      NOT an A/B write; recovery is UART/TFTP re-flash, not a reset-button hold.\n\n" +
	"Verify the image BEFORE planning: `quarry verify-ubi factory.ubi --board <name>`\n" +
	"confirms the FIT 'kernel' volume has a board-matched config node — the OpenWrt\n" +
	"port only boots from NAND when that matches (see docs/openwrt-ews377ap-v3.md).\n" +
	"Neither subcommand writes anything; the plan's steps are run by hand over UART.\n"
