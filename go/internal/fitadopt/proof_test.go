package fitadopt

import (
	"strings"
	"testing"
)

func goodExpected() Expected {
	return Expected{
		Model: "ap-hk07", FirmwareFamily: "fit", FitVersion: "1.1.65",
		RealSerial: "SWLWX420001Q", ActiveSlot: "rootfs_1",
	}
}

func goodObserved() Observed {
	return Observed{
		Model: "ap-hk07", FirmwareFamily: "fit", FitVersion: "1.1.65",
		RealSerial: "SWLWX420001Q", ActiveSlot: "rootfs_1",
		AccessOK: true, ServiceHealthy: true,
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
	p := Prove(goodExpected(), obs)
	if p.Passed {
		t.Fatal("a changed serial must fail the proof")
	}
	found := false
	for _, f := range p.Failures {
		if strings.Contains(f, "serial CHANGED") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a serial-changed failure, got %v", p.Failures)
	}
}

func TestProveEachCheck(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Observed)
	}{
		{"wrong model", func(o *Observed) { o.Model = "ecw230v3" }},
		{"not fit", func(o *Observed) { o.FirmwareFamily = "cloud" }},
		{"below floor", func(o *Observed) { o.FitVersion = "1.1.64" }},
		{"wrong version", func(o *Observed) { o.FitVersion = "1.2.0" }},
		{"wrong slot", func(o *Observed) { o.ActiveSlot = "rootfs" }},
		{"no access", func(o *Observed) { o.AccessOK = false }},
		{"unhealthy", func(o *Observed) { o.ServiceHealthy = false }},
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
	obs.AccessOK = false
	obs.ServiceHealthy = false
	p := Prove(goodExpected(), obs)
	if p.Passed || len(p.Failures) != 2 {
		t.Fatalf("expected 2 failures, got %v", p.Failures)
	}
}
