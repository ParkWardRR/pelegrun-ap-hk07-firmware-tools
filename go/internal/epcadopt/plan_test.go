package epcadopt

import (
	"strings"
	"testing"
)

func TestPlanRefusesIneligible(t *testing.T) {
	r := goodRequest()
	r.RecoveryRoutes = nil // no recovery route → ineligible
	if _, err := Plan(r); err == nil {
		t.Fatal("Plan must refuse an ineligible request")
	}
}

func TestPlanIsOrderedAndGated(t *testing.T) {
	steps, err := Plan(goodRequest())
	if err != nil {
		t.Fatalf("eligible request should plan: %v", err)
	}
	if len(steps) == 0 {
		t.Fatal("expected a non-empty plan")
	}

	// The backup gate must come before the first device-side mutation, so an
	// operator can never be told to point the AP away before backing it up.
	backup, point := -1, -1
	var gated int
	for i, st := range steps {
		if st.Gate {
			gated++
		}
		if strings.Contains(st.Desc, "backup") {
			backup = i
		}
		if strings.Contains(st.Desc, "point the AP at the controller") {
			point = i
		}
	}
	if backup == -1 || point == -1 {
		t.Fatalf("plan missing backup or point-at-controller step: %+v", steps)
	}
	if backup >= point {
		t.Fatalf("backup (%d) must precede pointing at the controller (%d)", backup, point)
	}
	if gated == 0 {
		t.Fatal("plan must have at least one hard gate")
	}
}

func TestPlanCarriesRealIdentityNotGenerated(t *testing.T) {
	r := goodRequest()
	steps, err := Plan(r)
	if err != nil {
		t.Fatal(err)
	}
	var joined string
	for _, st := range steps {
		joined += st.Desc + " " + st.Note + "\n"
	}
	if !strings.Contains(joined, r.RealSerial) {
		t.Fatalf("register step must reference the device's real serial %q", r.RealSerial)
	}
	if !strings.Contains(joined, r.ControllerAddr) {
		t.Fatalf("plan must reference the operator-supplied controller %q", r.ControllerAddr)
	}
}
