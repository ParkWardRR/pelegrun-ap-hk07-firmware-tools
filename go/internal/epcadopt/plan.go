package epcadopt

import (
	"fmt"
	"strings"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/flash"
)

// Plan builds the ordered, gated EPC-adoption plan (FR6). It reuses
// flash.Step{Desc,Note,Gate} so the CLI renders it with the exact same
// [GATE] loop as every other plan in this tool (see cmd/pelegrun/openwrt.go),
// rather than inventing a parallel step type.
//
// It refuses (error) if the Request fails the read-only eligibility gate —
// the same "don't hand back a plan for an operation you already know is
// ineligible" posture flash.Plan takes toward an incomplete env.
//
// The steps are deliberately transport-agnostic: they name WHAT must happen
// and in what order, not the controller-specific HTTP calls. T0.1/T0.2 are
// now resolved (a real REST API exists, org/hv/network scoped, two separate
// auth models for users vs. device checkin — see Client's doc comment), but
// Plan still doesn't hardcode the calls: Client's concrete implementation
// (Phase 2, still open) is what should actually drive them. Two notes are
// load-bearing:
//
//  1. Pointing the AP at the controller (jess.Cloud.SetForceAC) is a
//     device-side config mutation; per Constitution III it sits behind the
//     same backup gate a flash would, even though it is not itself a flash —
//     so "point at controller" is never treated as harmless just because no
//     partition is written.
//  2. A controller-side checkin gate has been observed to silently revert to
//     disabled (spec.md scenario 5). The plan re-asserts it as an explicit
//     step rather than assuming it stays where the operator last set it.
func Plan(r Request) ([]flash.Step, error) {
	if res := Validate(r); !res.Eligible {
		return nil, fmt.Errorf("not eligible for EPC adoption: %s", strings.Join(res.Reasons, "; "))
	}
	return []flash.Step{
		{Desc: "backup bundle before any device-side mutation", Note: "SetForceAC writes device config; back it up per Constitution III even though pointing at a controller is not a flash", Gate: true},
		{Desc: "confirm a recovery route is live", Note: fmt.Sprintf("at least one of %v reachable before pointing the AP away from its current management", r.RecoveryRoutes), Gate: true},
		{Desc: "point the AP at the controller", Note: fmt.Sprintf("jess.Cloud.SetForceAC → %s (device-side mutation)", r.ControllerAddr), Gate: false},
		{Desc: "verify the controller pointer persisted", Note: "re-read the AP's discovery override AFTER an AP reboot — a one-time read is not proof (spec 002 scenario 1/4)", Gate: true},
		{Desc: "re-assert the controller-side checkin gate", Note: "re-assert, do not assume: this setting has been observed to silently revert to disabled (spec 002 scenario 5)", Gate: false},
		{Desc: "register the device in controller inventory", Note: fmt.Sprintf("org=%s hv=%s network=%s, keyed by the device's REAL serial %s (never a generated identity)", r.OrgID, r.HVID, r.NetworkID, r.RealSerial), Gate: false},
		{Desc: "confirm checkin authenticated", Note: "controller reports auth success, not a silent/ambiguous 4xx (spec 002 FR5)", Gate: true},
		{Desc: "prove adoption survives disruption", Note: "adopted state must be intact after BOTH an AP reboot and a controller restart, or the plan documents why that check was skipped (epcadopt.Prove)", Gate: true},
	}, nil
}
