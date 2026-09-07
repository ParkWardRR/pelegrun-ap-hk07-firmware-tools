package epcadopt

import (
	"fmt"
	"sort"
	"strings"
)

// Expected is the intended post-adoption state, taken from the immutable plan.
type Expected struct {
	Model          string `json:"model"`
	RealSerial     string `json:"real_serial"`     // the device's ORIGINAL serial — adoption must preserve it
	RealMAC        string `json:"real_mac"`        // the device's ORIGINAL MAC — adoption must preserve it
	ControllerAddr string `json:"controller_addr"` // the controller adoption should have pointed at
}

// Observed is what a re-probe (eyas/jess + a controller-side status read)
// actually found after an adoption attempt.
type Observed struct {
	Model               string `json:"model"`
	RealSerial          string `json:"real_serial"`
	RealMAC             string `json:"real_mac"`
	ControllerAddr      string `json:"controller_addr"`       // what the device itself reports as its controller pointer
	ControllerPointed   bool   `json:"controller_pointed"`    // pointer setting persisted across a reboot
	Reachable           bool   `json:"reachable"`             // AP can reach the controller address
	AuthOK              bool   `json:"auth_ok"`               // checkin authenticated (not a 4xx/silent failure)
	InventoryMatch      bool   `json:"inventory_match"`       // controller inventory shows this exact serial+MAC
	Adopted             bool   `json:"adopted"`               // controller reports this device as adopted
	SurvivedAPReboot    bool   `json:"survived_ap_reboot"`    // adoption state intact after an AP reboot
	SurvivedCtrlRestart bool   `json:"survived_ctrl_restart"` // adoption state intact after a controller restart
}

// Proof is the post-adoption verdict. A successful registration call is NOT
// success — adoption is only proven when identity, controller pointer,
// reachability, auth, inventory match, adopted status, and durability across
// both an AP reboot and a controller restart all check out. Mirrors
// fitadopt.Proof's shape and "report every failure" posture exactly.
type Proof struct {
	Passed   bool
	Failures []string
}

// Prove verifies observed state against the intended plan, reporting every
// failure rather than stopping at the first. The identity check is the
// load-bearing one: if the observed serial/MAC differ from the expected
// originals, adoption did NOT preserve real identity and the result must be
// rejected regardless of everything else — same rule fitadopt.Prove enforces
// for FIT.
func Prove(exp Expected, obs Observed) Proof {
	var fail []string

	if exp.Model != "" && !strings.EqualFold(exp.Model, obs.Model) {
		fail = append(fail, fmt.Sprintf("model: expected %q, observed %q", exp.Model, obs.Model))
	}

	if exp.RealSerial == "" || obs.RealSerial == "" {
		fail = append(fail, "real serial missing from expected or observed state")
	} else if !strings.EqualFold(exp.RealSerial, obs.RealSerial) {
		fail = append(fail, fmt.Sprintf("real serial CHANGED: expected %q, observed %q (adoption must preserve it)", exp.RealSerial, obs.RealSerial))
	}
	if exp.RealMAC == "" || obs.RealMAC == "" {
		fail = append(fail, "real MAC missing from expected or observed state")
	} else if !strings.EqualFold(exp.RealMAC, obs.RealMAC) {
		fail = append(fail, fmt.Sprintf("real MAC CHANGED: expected %q, observed %q (adoption must preserve it)", exp.RealMAC, obs.RealMAC))
	}

	if exp.ControllerAddr != "" && !strings.EqualFold(exp.ControllerAddr, obs.ControllerAddr) {
		fail = append(fail, fmt.Sprintf("controller pointer: expected %q, observed %q", exp.ControllerAddr, obs.ControllerAddr))
	}
	if !obs.ControllerPointed {
		fail = append(fail, "controller pointer did not persist")
	}
	if !obs.Reachable {
		fail = append(fail, "AP cannot reach the controller address")
	}
	if !obs.AuthOK {
		fail = append(fail, "checkin did not authenticate successfully")
	}
	if !obs.InventoryMatch {
		fail = append(fail, "controller inventory does not show this device's serial+MAC")
	}
	if !obs.Adopted {
		fail = append(fail, "controller does not report this device as adopted")
	}
	if !obs.SurvivedAPReboot {
		fail = append(fail, "adoption did not survive an AP reboot")
	}
	if !obs.SurvivedCtrlRestart {
		fail = append(fail, "adoption did not survive a controller restart")
	}

	sort.Strings(fail)
	return Proof{Passed: len(fail) == 0, Failures: fail}
}
