package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/fleet"
)

// cmdFleet is the P9 fleet control plane: `plan` builds a deterministic,
// read-only rollout plan (it changes no device); `apply` revalidates a saved plan
// against the live inventory/policy/target and reports whether it is safe to run.
// Execution through a jess accessor is intentionally not wired yet — the ROADMAP
// says to land the planner and guard first, "not batch flashing."
func cmdFleet(out io.Writer, a []string) error {
	if len(a) == 0 {
		fmt.Fprint(out, fleetUsage)
		return nil
	}
	sub, rest := a[0], a[1:]
	switch sub {
	case "plan":
		return cmdFleetPlan(out, rest)
	case "apply":
		return cmdFleetApply(out, rest)
	case "-h", "--help", "help":
		fmt.Fprint(out, fleetUsage)
		return nil
	default:
		return fmt.Errorf("fleet: unknown subcommand %q (try `swallow fleet help`)", sub)
	}
}

func loadPolicy(path string) (fleet.Policy, error) {
	var p fleet.Policy
	if path == "" {
		return p, fmt.Errorf("--policy <file.json> required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("policy %s: %w", path, err)
	}
	return p, nil
}

func targetFromArgs(a []string) (fleet.Target, error) {
	t := fleet.Target{
		ImageName:      argVal(a, "--image"),
		ImageSHA256:    argVal(a, "--image-sha256"),
		FirmwareFamily: argVal(a, "--family"),
		FitVersion:     argVal(a, "--fit-version"),
	}
	if t.ImageName == "" || t.ImageSHA256 == "" {
		return t, fmt.Errorf("--image <name> and --image-sha256 <hex> required")
	}
	return t, nil
}

// nowFromArgs allows a fixed --now RFC3339 for reproducible runs/tests; it
// defaults to the current UTC time.
func nowFromArgs(a []string) (time.Time, error) {
	if v := argVal(a, "--now"); v != "" {
		return time.Parse(time.RFC3339, v)
	}
	return time.Now().UTC(), nil
}

func cmdFleetPlan(out io.Writer, a []string) error {
	f, err := fleet.Load(argVal(a, "--inventory"))
	if err != nil {
		return fmt.Errorf("inventory: %w", err)
	}
	pol, err := loadPolicy(argVal(a, "--policy"))
	if err != nil {
		return err
	}
	tgt, err := targetFromArgs(a)
	if err != nil {
		return err
	}
	now, err := nowFromArgs(a)
	if err != nil {
		return fmt.Errorf("--now: %w", err)
	}

	pl := fleet.BuildPlan(f, pol, tgt, Version, now)
	if outPath := argVal(a, "--out"); outPath != "" {
		if err := pl.Save(outPath); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
	}
	printPlan(out, pl)
	return nil
}

func printPlan(out io.Writer, pl *fleet.Plan) {
	fmt.Fprintf(out, "plan %s  target=%s (%s)\n", short(pl.PlanHash), pl.Target.ImageName, short(pl.Target.ImageSHA256))
	fmt.Fprintf(out, "policy=%s  window_open=%t  eligible=%d blocked=%d canary=%d\n",
		pl.PolicyVersion, pl.WindowOpen, pl.EligibleCount, pl.BlockedCount, pl.CanaryCount)
	for _, e := range pl.Entries {
		if e.Eligible {
			fmt.Fprintf(out, "  ✓ %-16s [%s] %s -> %s\n", e.FleetID, e.Cohort, e.ActiveSlot, e.InactiveSlot)
		} else {
			sort.Strings(e.Reasons)
			fmt.Fprintf(out, "  ✗ %-16s blocked: %v\n", e.FleetID, e.Reasons)
		}
	}
}

func cmdFleetApply(out io.Writer, a []string) error {
	planPath := argVal(a, "--plan")
	if planPath == "" {
		return fmt.Errorf("--plan <file.json> required")
	}
	pl, err := fleet.LoadPlan(planPath)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	f, err := fleet.Load(argVal(a, "--inventory"))
	if err != nil {
		return fmt.Errorf("inventory: %w", err)
	}
	pol, err := loadPolicy(argVal(a, "--policy"))
	if err != nil {
		return err
	}
	tgt, err := targetFromArgs(a)
	if err != nil {
		return err
	}
	now, err := nowFromArgs(a)
	if err != nil {
		return fmt.Errorf("--now: %w", err)
	}
	maxAge := time.Duration(0)
	if v := argVal(a, "--max-age"); v != "" {
		if maxAge, err = time.ParseDuration(v); err != nil {
			return fmt.Errorf("--max-age: %w", err)
		}
	}

	if err := fleet.ApplyGuard(pl, f, pol, tgt, now, maxAge); err != nil {
		return fmt.Errorf("apply refused: %w", err)
	}
	fmt.Fprintf(out, "apply guard PASSED for plan %s\n", short(pl.PlanHash))
	fmt.Fprintf(out, "  %d eligible device(s), %d in the canary cohort\n", pl.EligibleCount, pl.CanaryCount)
	fmt.Fprintln(out, "  execution is not wired yet (no jess accessor); no device was changed.")
	return nil
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

const fleetUsage = "swallow fleet — policy- and evidence-driven batch rollout (P9)\n\n" +
	"USAGE:\n" +
	"  swallow fleet plan  --inventory inv.json --policy policy.json \\\n" +
	"        --image <name> --image-sha256 <hex> [--family cloud] [--fit-version x] [--out plan.json]\n" +
	"  swallow fleet apply --inventory inv.json --policy policy.json \\\n" +
	"        --image <name> --image-sha256 <hex> --plan plan.json [--max-age 10m]\n\n" +
	"plan is read-only (changes nothing). apply revalidates the saved plan against the\n" +
	"live inventory/policy/target and fails closed on any drift; it does not yet execute.\n"
