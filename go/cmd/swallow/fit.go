package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/fitadopt"
)

// cmdFit exposes the P10 FIT real-serial gates: `check` runs the read-only
// adoption-eligibility validator; `prove` runs the post-adoption verification.
// Both are read-only and change nothing on any device.
func cmdFit(out io.Writer, a []string) error {
	sub := ""
	if len(a) > 0 {
		sub = a[0]
		a = a[1:]
	}
	switch sub {
	case "check":
		return cmdFitCheck(out, a)
	case "prove":
		return cmdFitProve(out, a)
	case "", "-h", "--help", "help":
		fmt.Fprint(out, fitUsage)
		return nil
	default:
		return fmt.Errorf("fit: unknown subcommand %q (check|prove)", sub)
	}
}

func loadJSON(path string, v any) error {
	if path == "" {
		return fmt.Errorf("missing required --request/--expected/--observed file")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func cmdFitCheck(out io.Writer, a []string) error {
	var req fitadopt.Request
	if err := loadJSON(argVal(a, "--request"), &req); err != nil {
		return fmt.Errorf("fit check: %w", err)
	}
	res := fitadopt.Validate(req)
	if res.Eligible {
		fmt.Fprintf(out, "ELIGIBLE — %s may adopt via FIT %s (serial %s preserved)\n", req.Model, req.FitVersion, req.RealSerial)
		return nil
	}
	fmt.Fprintln(out, "NOT ELIGIBLE:")
	for _, r := range res.Reasons {
		fmt.Fprintf(out, "  - %s\n", r)
	}
	return fmt.Errorf("%d eligibility problem(s)", len(res.Reasons))
}

func cmdFitProve(out io.Writer, a []string) error {
	var exp fitadopt.Expected
	var obs fitadopt.Observed
	if err := loadJSON(argVal(a, "--expected"), &exp); err != nil {
		return fmt.Errorf("fit prove --expected: %w", err)
	}
	if err := loadJSON(argVal(a, "--observed"), &obs); err != nil {
		return fmt.Errorf("fit prove --observed: %w", err)
	}
	p := fitadopt.Prove(exp, obs)
	if p.Passed {
		fmt.Fprintf(out, "PROVEN — adoption verified for %s (serial %s preserved, FIT %s)\n", exp.Model, exp.RealSerial, obs.FitVersion)
		return nil
	}
	fmt.Fprintln(out, "NOT PROVEN:")
	for _, f := range p.Failures {
		fmt.Fprintf(out, "  - %s\n", f)
	}
	return fmt.Errorf("%d proof failure(s)", len(p.Failures))
}

const fitUsage = "swallow fit — P10 FIT real-serial adoption gates (read-only)\n\n" +
	"USAGE:\n" +
	"  swallow fit check --request req.json                 eligibility (P10a): may this device adopt?\n" +
	"  swallow fit prove --expected exp.json --observed obs.json   post-adoption proof (P10b)\n\n" +
	"check enforces FIT >= " + fitadopt.MinFitVersion + ", a hashed+provenanced image, a recovery route, and a\n" +
	"real (never generated/spoofed) serial. prove verifies the serial was preserved,\n" +
	"the version/slot/identity match, and access + service health after adoption.\n"
