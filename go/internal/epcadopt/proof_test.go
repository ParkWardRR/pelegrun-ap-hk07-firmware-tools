package epcadopt

import "testing"

func goodExpected() Expected {
	return Expected{
		Model: "ap-hk07", RealSerial: "SWLWX420001T", RealMAC: "AA:BB:CC:11:22:33",
		ControllerAddr: "controller.example",
	}
}

func goodObserved() Observed {
	return Observed{
		Model: "ap-hk07", RealSerial: "SWLWX420001T", RealMAC: "AA:BB:CC:11:22:33",
		ControllerAddr: "controller.example", ControllerPointed: true, Reachable: true,
		AuthOK: true, InventoryMatch: true, Adopted: true,
		SurvivedAPReboot: true, SurvivedCtrlRestart: true,
	}
}

func TestProvePasses(t *testing.T) {
	if p := Prove(goodExpected(), goodObserved()); !p.Passed {
		t.Fatalf("expected proof to pass, failures: %v", p.Failures)
	}
}

func TestProveSerialChangedFails(t *testing.T) {
	obs := goodObserved()
	obs.RealSerial = "SWLWX420002N"
	if Prove(goodExpected(), obs).Passed {
		t.Fatal("a changed serial must fail the proof")
	}
}

func TestProveEachCheck(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Observed)
	}{
		{"wrong model", func(o *Observed) { o.Model = "ecw230v3" }},
		{"mac changed", func(o *Observed) { o.RealMAC = "FF:FF:FF:FF:FF:FF" }},
		{"wrong controller", func(o *Observed) { o.ControllerAddr = "other.example" }},
		{"pointer not persisted", func(o *Observed) { o.ControllerPointed = false }},
		{"unreachable", func(o *Observed) { o.Reachable = false }},
		{"auth failed", func(o *Observed) { o.AuthOK = false }},
		{"no inventory match", func(o *Observed) { o.InventoryMatch = false }},
		{"not adopted", func(o *Observed) { o.Adopted = false }},
		{"reboot not survived", func(o *Observed) { o.SurvivedAPReboot = false }},
		{"restart not survived", func(o *Observed) { o.SurvivedCtrlRestart = false }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := goodObserved()
			tc.mut(&obs)
			if Prove(goodExpected(), obs).Passed {
				t.Fatalf("%s should fail the proof", tc.name)
			}
		})
	}
}

func TestProveReportsAllFailures(t *testing.T) {
	obs := goodObserved()
	obs.AuthOK = false
	obs.Adopted = false
	p := Prove(goodExpected(), obs)
	if p.Passed || len(p.Failures) != 2 {
		t.Fatalf("expected 2 failures, got %v", p.Failures)
	}
}
