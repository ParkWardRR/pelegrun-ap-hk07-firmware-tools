package fitadopt

import (
	"fmt"
	"sort"
	"strings"
)

// Expected is the intended post-adoption state, taken from the immutable plan.
type Expected struct {
	Model          string
	FirmwareFamily string // "fit"
	FitVersion     string // the version the device should now report
	RealSerial     string // the device's ORIGINAL serial — adoption must preserve it
	ActiveSlot     string // the slot adoption should have activated
}

// Observed is what a re-probe (eyas/jess) actually found after adoption.
type Observed struct {
	Model          string
	FirmwareFamily string
	FitVersion     string
	RealSerial     string
	ActiveSlot     string
	AccessOK       bool // the intended jess adapter reconnected and authenticated
	ServiceHealthy bool // a model-appropriate health check passed (not just ping)
}

// Proof is the P10b post-adoption verdict. A successful command or reboot is NOT
// success — adoption is only proven when identity, version, the preserved serial,
// the boot slot, access, and service health all check out.
type Proof struct {
	Passed   bool
	Failures []string
}

// Prove verifies observed state against the intended plan. It reports every
// failure, not just the first. The serial check is the load-bearing one: if the
// observed serial differs from the expected original, adoption did NOT preserve
// the real serial and the result must be rejected regardless of everything else.
func Prove(exp Expected, obs Observed) Proof {
	var fail []string

	if exp.Model != "" && !strings.EqualFold(exp.Model, obs.Model) {
		fail = append(fail, fmt.Sprintf("model: expected %q, observed %q", exp.Model, obs.Model))
	}
	if !strings.EqualFold(obs.FirmwareFamily, "fit") {
		fail = append(fail, fmt.Sprintf("firmware family: expected FIT, observed %q", obs.FirmwareFamily))
	}
	if exp.FirmwareFamily != "" && !strings.EqualFold(exp.FirmwareFamily, obs.FirmwareFamily) {
		fail = append(fail, fmt.Sprintf("firmware family: expected %q, observed %q", exp.FirmwareFamily, obs.FirmwareFamily))
	}

	switch cmp, err := CompareVersions(obs.FitVersion, MinFitVersion); {
	case err != nil:
		fail = append(fail, fmt.Sprintf("FIT version %q is unparseable", obs.FitVersion))
	case cmp < 0:
		fail = append(fail, fmt.Sprintf("FIT version %s is below the supported floor %s", obs.FitVersion, MinFitVersion))
	}
	if exp.FitVersion != "" {
		if cmp, err := CompareVersions(obs.FitVersion, exp.FitVersion); err != nil || cmp != 0 {
			fail = append(fail, fmt.Sprintf("FIT version: expected %s, observed %s", exp.FitVersion, obs.FitVersion))
		}
	}

	// The real serial must be unchanged. This is the non-negotiable check.
	if exp.RealSerial == "" || obs.RealSerial == "" {
		fail = append(fail, "real serial missing from expected or observed state")
	} else if !strings.EqualFold(exp.RealSerial, obs.RealSerial) {
		fail = append(fail, fmt.Sprintf("real serial CHANGED: expected %q, observed %q (adoption must preserve it)", exp.RealSerial, obs.RealSerial))
	}

	if exp.ActiveSlot != "" && !strings.EqualFold(exp.ActiveSlot, obs.ActiveSlot) {
		fail = append(fail, fmt.Sprintf("active slot: expected %q, observed %q", exp.ActiveSlot, obs.ActiveSlot))
	}
	if !obs.AccessOK {
		fail = append(fail, "management access did not reconnect/authenticate after adoption")
	}
	if !obs.ServiceHealthy {
		fail = append(fail, "post-adoption service health check did not pass")
	}

	sort.Strings(fail)
	return Proof{Passed: len(fail) == 0, Failures: fail}
}
