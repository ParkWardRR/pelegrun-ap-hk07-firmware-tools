package fleet

import (
	"testing"
	"time"
)

var refNow = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

func verifiedDevice() Device {
	return Device{
		FleetID:  "a",
		Physical: PhysicalIdentity{Model: "ap-hk07", HardwareSerial: "SWLWX420001Q", Confidence: ConfidenceVerified},
		Software: SoftwareIdentity{FirmwareFamily: "cloud", ActiveSlot: "rootfs", InactiveSlot: "rootfs_1"},
		Access:   Access{ManagementAddress: "192.168.1.10", LastSeenAt: refNow.Format(time.RFC3339)},
		Evidence: Evidence{BackupBundleSHA256: "abc"},
	}
}

func basePolicy() Policy {
	return Policy{
		Version:           "v1",
		SupportedModels:   []string{"ap-hk07"},
		AllowedFamilies:   []string{"cloud", "ews-luci"},
		RequireBackup:     true,
		StaleAfterSeconds: 600,
	}
}

func TestEvaluateEligibleHappyPath(t *testing.T) {
	dec := basePolicy().Evaluate(verifiedDevice(), Target{}, refNow)
	if !dec.Eligible {
		t.Fatalf("expected eligible, blocked by %v", dec.Reasons)
	}
}

func TestEvaluateBlocksEachRule(t *testing.T) {
	cases := []struct {
		name  string
		mutd  func(*Device)
		wantN int
	}{
		{"conflict identity", func(d *Device) { d.Physical.Confidence = ConfidenceConflict }, 1},
		{"low confidence", func(d *Device) { d.Physical.Confidence = ConfidenceProbable }, 1},
		{"wrong model", func(d *Device) { d.Physical.Model = "ecw230v3" }, 1},
		{"wrong family", func(d *Device) { d.Software.FirmwareFamily = "fit" }, 1},
		{"no backup", func(d *Device) { d.Evidence.BackupBundleSHA256 = "" }, 1},
		{"stale", func(d *Device) { d.Access.LastSeenAt = refNow.Add(-time.Hour).Format(time.RFC3339) }, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := verifiedDevice()
			tc.mutd(&d)
			dec := basePolicy().Evaluate(d, Target{}, refNow)
			if dec.Eligible {
				t.Fatalf("%s should block", tc.name)
			}
			if len(dec.Reasons) != tc.wantN {
				t.Fatalf("%s: want %d reasons, got %v", tc.name, tc.wantN, dec.Reasons)
			}
		})
	}
}

func TestEvaluateReportsAllBlocks(t *testing.T) {
	d := verifiedDevice()
	d.Physical.Model = "ecw230v3"
	d.Evidence.BackupBundleSHA256 = ""
	dec := basePolicy().Evaluate(d, Target{}, refNow)
	if dec.Eligible || len(dec.Reasons) != 2 {
		t.Fatalf("expected 2 blocking reasons, got %v", dec.Reasons)
	}
}

func TestPolicyHashStableAndSensitive(t *testing.T) {
	p := basePolicy()
	if p.Hash() != basePolicy().Hash() {
		t.Fatal("equal policies must hash equally")
	}
	p2 := basePolicy()
	p2.RequireBackup = false
	if p.Hash() == p2.Hash() {
		t.Fatal("a policy change must change the hash")
	}
}

func TestWindowOpen(t *testing.T) {
	var none *Window
	if !none.Open(refNow) {
		t.Fatal("nil window is always open")
	}
	w := &Window{StartRFC3339: "2026-09-04T10:00:00Z", EndRFC3339: "2026-09-04T14:00:00Z"}
	if !w.Open(refNow) {
		t.Fatal("noon is inside 10-14")
	}
	if w.Open(refNow.Add(4 * time.Hour)) {
		t.Fatal("16:00 is outside 10-14")
	}
	bad := &Window{StartRFC3339: "not-a-time"}
	if bad.Open(refNow) {
		t.Fatal("unparseable bound must fail closed")
	}
}
