// Package epcadopt is the pairing/adoption gate for bringing an ap-hk07 AP
// under a self-hosted EnGenius Private Cloud (EPC) controller — the
// self-hosted-controller sibling of `fitadopt`'s FitController real-serial
// path. See specs/002-epc-pairing/{spec,plan,tasks}.md for the full design.
//
// FRAMEWORK ONLY as of this writing: several fields and checks below are
// marked TODO because they depend on protocol facts that specs/002's "Open
// questions" section says must be confirmed against a real controller
// instance before they can be implemented correctly — see
// specs/002-epc-pairing/tasks.md Phase 0. Do not guess a plausible-looking
// value for any of them (a specific version-support claim that was never
// actually checked against a real vendor artifact is exactly the mistake
// `fitadopt.MinFitVersion` had to be corrected for — see that constant's own
// doc comment in eligibility.go of the fitadopt package).
//
// Absolute rule, unchanged from fitadopt: adoption preserves the device's
// EXISTING real serial and MAC. This package never generates, reuses, or
// spoofs either.
package epcadopt

import (
	"fmt"
	"strings"
)

// Request is a proposed EPC adoption, assembled read-only from device
// inspection (eyas/jess) and the operator's chosen controller scope. Every
// field here is either read from the device or supplied explicitly by the
// operator — never a hardcoded default (Constitution VII: no real
// controller/org/network identity ever lives in this codebase).
//
// Scope confirmed against a real controller's own OpenAPI schema (T0.2):
// device registration is scoped org -> hierarchy view (hv) -> network, a
// three-level hierarchy, not the two-level org/network this type originally
// assumed. HVID is that middle "hierarchy view" scope — e.g.
// `/api/v1/orgs/{org_id}/hvs/{hv_id}/networks/{network_id}/devices`.
type Request struct {
	Model          string   `json:"model"`           // device model, e.g. ap-hk07
	FirmwareFamily string   `json:"firmware_family"` // eyas family string; which families are eligible is TODO, see T0.3
	RealSerial     string   `json:"real_serial"`     // serial READ FROM the device — never generated
	RealMAC        string   `json:"real_mac"`        // MAC READ FROM the device — never generated
	ControllerAddr string   `json:"controller_addr"` // operator-supplied controller address; never defaulted
	OrgID          string   `json:"org_id"`          // operator-supplied controller org scope
	HVID           string   `json:"hv_id"`           // operator-supplied "hierarchy view" scope, between org and network
	NetworkID      string   `json:"network_id"`      // operator-supplied controller network scope
	RecoveryRoutes []string `json:"recovery_routes"` // e.g. ["ab-rollback","uart","tftp"] — at least one
}

// Result is the eligibility outcome. Reasons list every block so an operator
// sees the whole picture, not just the first failure — same shape as
// fitadopt.Result.
type Result struct {
	Eligible bool
	Reasons  []string
}

// Validate applies every EPC-adoption precondition that can be checked
// without a live controller connection. It is deterministic and side-effect
// free; a true result means the operation is *allowed to be planned*, not
// that it will succeed — post-adoption proof is a separate gate (see
// Prove in proof.go).
//
// TODO(epc, T0.3): once a real controller instance confirms which firmware
// families/models it accepts for adoption, add that check here — do not add
// it as a guessed allow-list before then. Until resolved, FirmwareFamily is
// recorded but not gated on.
func Validate(r Request) Result {
	var reasons []string

	if r.Model == "" {
		reasons = append(reasons, "device model is unknown; refuse adoption without a model/layout match")
	}
	if strings.TrimSpace(r.ControllerAddr) == "" {
		reasons = append(reasons, "no controller address supplied (must be explicit operator input, never defaulted)")
	}
	if strings.TrimSpace(r.OrgID) == "" {
		reasons = append(reasons, "no controller org scope supplied")
	}
	if strings.TrimSpace(r.HVID) == "" {
		reasons = append(reasons, "no controller hierarchy-view (hv) scope supplied")
	}
	if strings.TrimSpace(r.NetworkID) == "" {
		reasons = append(reasons, "no controller network scope supplied")
	}

	if err := validateRealIdentity(r.RealSerial, r.RealMAC); err != nil {
		reasons = append(reasons, err.Error())
	}

	if len(r.RecoveryRoutes) == 0 {
		reasons = append(reasons, "no recovery route established (need at least one of ab-rollback/uart/tftp)")
	}

	return Result{Eligible: len(reasons) == 0, Reasons: reasons}
}

// validateRealIdentity enforces that the serial and MAC look like device-read
// identity, not placeholders or generated values — the same posture
// fitadopt.validateRealSerial takes for FIT adoption.
func validateRealIdentity(serial, mac string) error {
	serial = strings.TrimSpace(serial)
	mac = strings.TrimSpace(mac)
	if serial == "" {
		return fmt.Errorf("no real serial read from the device (adoption must preserve the existing serial)")
	}
	if mac == "" {
		return fmt.Errorf("no real MAC read from the device (adoption must preserve the existing MAC)")
	}
	lower := strings.ToLower(serial + " " + mac)
	for _, bad := range []string{"spoof", "fake", "generated", "placeholder", "000000000000", "00:00:00:00:00:00"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("identity %q/%q looks generated/placeholder; only real device-read identity is allowed", serial, mac)
		}
	}
	return nil
}
