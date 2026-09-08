package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/epcadopt"
)

// epcModel is the registry/board key these EPC-adoption gates default to —
// the same physical ap-hk07 board every other command in this tool targets.
// Override with --model if a different board is ever brought under this path.
const epcModel = "ap-hk07"

// cmdEpc exposes the spec-002 EPC (EnGenius Private Cloud) pairing gates:
// `check` is the read-only AP-identity + eligibility report (FR4), `plan`
// prints the ordered, gated adoption plan (FR6), and `prove` verifies a
// post-adoption observation against its intended plan (FR5), mirroring
// `pelegrun fit prove` exactly.
//
// None of the three mutates a device or a controller. The actual controller
// client (login/register/status) is intentionally NOT wired here yet: its
// API/auth shape is an open question that must be confirmed against a real
// controller first (specs/002-epc-pairing Phase 0). `check`/`plan` therefore
// report the AP-side and eligibility facts they CAN establish honestly, and
// name the controller-side verification as pending rather than faking it.
func cmdEpc(out io.Writer, a []string) error {
	sub := ""
	if len(a) > 0 {
		sub = a[0]
		a = a[1:]
	}
	switch sub {
	case "check":
		return cmdEpcCheck(out, a)
	case "plan":
		return cmdEpcPlan(out, a)
	case "prove":
		return cmdEpcProve(out, a)
	case "", "-h", "--help", "help":
		fmt.Fprint(out, epcUsage)
		return nil
	default:
		return fmt.Errorf("epc: unknown subcommand %q (check|plan|prove)", sub)
	}
}

// epcRequest reads the AP's real identity live (reusing the same cloud-firmware
// login+SysInfo path as crossflash) and assembles a Request from that identity
// plus the operator-supplied controller scope. Every controller-identifying
// value is an explicit flag, never a default (Constitution VII).
func epcRequest(a []string) (epcadopt.Request, error) {
	c, ap, err := crossflashClient(a)
	if err != nil {
		return epcadopt.Request{}, err
	}
	info, err := c.SysInfo(context.Background())
	if err != nil {
		return epcadopt.Request{}, fmt.Errorf("sys_info %s: %w", ap, err)
	}
	var routes []string
	if r := strings.TrimSpace(argVal(a, "--recovery")); r != "" {
		for _, part := range strings.Split(r, ",") {
			if p := strings.TrimSpace(part); p != "" {
				routes = append(routes, p)
			}
		}
	}
	return epcadopt.Request{
		Model:          or(argVal(a, "--model"), epcModel),
		FirmwareFamily: argVal(a, "--family"),
		RealSerial:     info.Serial,
		RealMAC:        info.MAC,
		ControllerAddr: argVal(a, "--controller"),
		OrgID:          argVal(a, "--org"),
		HVID:           argVal(a, "--hv"),
		NetworkID:      argVal(a, "--network"),
		RecoveryRoutes: routes,
	}, nil
}

func cmdEpcCheck(out io.Writer, a []string) error {
	if strings.TrimSpace(argVal(a, "--controller")) == "" {
		return fmt.Errorf("epc check: --controller <addr> required (never defaulted, per Constitution VII)")
	}
	req, err := epcRequest(a)
	if err != nil {
		return fmt.Errorf("epc check: %w", err)
	}
	fmt.Fprintf(out, "AP identity (read live):\n  serial : %s\n  mac    : %s\n", req.RealSerial, req.RealMAC)
	res := epcadopt.Validate(req)
	if res.Eligible {
		fmt.Fprintf(out, "ELIGIBLE — %s may be planned for EPC adoption at %s (serial %s preserved)\n", req.Model, req.ControllerAddr, req.RealSerial)
	} else {
		fmt.Fprintln(out, "NOT ELIGIBLE:")
		for _, r := range res.Reasons {
			fmt.Fprintf(out, "  - %s\n", r)
		}
	}
	fmt.Fprint(out, "\ncontroller reachability/auth: NOT VERIFIED — the controller client is\n"+
		"pending confirmation of its real API/auth shape against a live instance\n"+
		"(specs/002-epc-pairing Phase 0); this check reports AP-side facts only.\n")
	if !res.Eligible {
		return fmt.Errorf("%d eligibility problem(s)", len(res.Reasons))
	}
	return nil
}

func cmdEpcPlan(out io.Writer, a []string) error {
	req, err := epcRequest(a)
	if err != nil {
		return fmt.Errorf("epc plan: %w", err)
	}
	steps, err := epcadopt.Plan(req)
	if err != nil {
		return fmt.Errorf("epc plan: %w — run `pelegrun epc check` first", err)
	}
	for i, st := range steps {
		gate := ""
		if st.Gate {
			gate = " [GATE]"
		}
		fmt.Fprintf(out, "%d. %s%s\n    %s\n", i+1, st.Desc, gate, st.Note)
	}
	fmt.Fprint(out, "\nplan only — nothing mutated. The controller-side steps (register/checkin)\n"+
		"have no automated apply yet; they await a confirmed controller client\n"+
		"(specs/002-epc-pairing Phase 0/2). Run them by hand for now.\n")
	return nil
}

func cmdEpcProve(out io.Writer, a []string) error {
	var exp epcadopt.Expected
	var obs epcadopt.Observed
	if err := loadJSON(argVal(a, "--expected"), &exp); err != nil {
		return fmt.Errorf("epc prove --expected: %w", err)
	}
	if err := loadJSON(argVal(a, "--observed"), &obs); err != nil {
		return fmt.Errorf("epc prove --observed: %w", err)
	}
	p := epcadopt.Prove(exp, obs)
	if p.Passed {
		fmt.Fprintf(out, "PROVEN — EPC adoption verified for %s (serial %s preserved, controller %s)\n", exp.Model, exp.RealSerial, obs.ControllerAddr)
		return nil
	}
	fmt.Fprintln(out, "NOT PROVEN:")
	for _, f := range p.Failures {
		fmt.Fprintf(out, "  - %s\n", f)
	}
	return fmt.Errorf("%d proof failure(s)", len(p.Failures))
}

const epcUsage = "pelegrun epc — spec-002 EnGenius Private Cloud pairing gates (read-only)\n\n" +
	"USAGE:\n" +
	"  pelegrun epc check --ap <ip> --controller <addr> [--org ID] [--hv ID] [--network ID]\n" +
	"                     [--recovery ab-rollback,uart] [--user admin] [--pass admin]\n" +
	"      read AP identity live + report FR4 eligibility. Controller auth is not\n" +
	"      verified yet (protocol pending, specs/002 Phase 0).\n" +
	"  pelegrun epc plan  --ap <ip> --controller <addr> --org ID --hv ID --network ID\n" +
	"                     [--recovery ab-rollback,uart]\n" +
	"      ordered, gated adoption plan (FR6) — point at controller, register,\n" +
	"      prove checkin. Prints only; nothing is mutated.\n" +
	"  pelegrun epc prove --expected exp.json --observed obs.json\n" +
	"      post-adoption proof (FR5), modeled on `pelegrun fit prove`.\n\n" +
	"Controller scope is three levels: org -> hv (hierarchy view) -> network.\n" +
	"Every controller-identifying value (--controller/--org/--hv/--network/creds) is\n" +
	"explicit operator input, never a default (Constitution VII). Adoption always\n" +
	"preserves the device's REAL serial/MAC; this tool never generates either.\n"
