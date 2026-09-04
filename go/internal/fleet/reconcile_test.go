package fleet

import (
	"testing"
	"time"
)

func TestReconcileUpdatesSoftwareAndVerifies(t *testing.T) {
	seen := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	prev := Device{
		FleetID:  "a",
		Physical: PhysicalIdentity{Model: "ap-hk07", HardwareSerial: "SWLWX420001Q", Confidence: ConfidenceProbable},
		Software: SoftwareIdentity{ActiveSlot: "rootfs", FirmwareFamily: "cloud"},
	}
	obs := Observed{
		FleetID: "a", Model: "ap-hk07", HardwareSerial: "SWLWX420001Q",
		ActiveSlot: "rootfs_1", FirmwareFamily: "cloud", FirmwareVersion: "3.0.1",
		ManagementAddress: "192.168.1.10", SeenAt: seen,
	}
	r := Reconcile(prev, obs)
	if r.Mismatch {
		t.Fatalf("matching serial must not be a mismatch: %v", r.MismatchReasons)
	}
	if r.Device.Physical.Confidence != ConfidenceVerified {
		t.Fatalf("matched hardware serial should verify identity, got %s", r.Device.Physical.Confidence)
	}
	if r.Device.Software.ActiveSlot != "rootfs_1" {
		t.Fatal("active slot should update to the observed value")
	}
	if r.Device.Access.LastSeenAt != seen.Format(time.RFC3339) {
		t.Fatal("last-seen should be refreshed")
	}
	// Only the slot is a real change: it had a prior value. The firmware version
	// fills a previously-empty field (not a change), and the family is unchanged.
	if len(r.Changed) != 1 {
		t.Fatalf("expected 1 recorded software change, got %v", r.Changed)
	}
}

func TestReconcileFlagsIdentityMismatch(t *testing.T) {
	prev := Device{
		FleetID:  "a",
		Physical: PhysicalIdentity{Model: "ap-hk07", HardwareSerial: "SWLWX420001Q", Confidence: ConfidenceVerified},
	}
	obs := Observed{FleetID: "a", Model: "ap-hk07", HardwareSerial: "OTHER0000009"}
	r := Reconcile(prev, obs)
	if !r.Mismatch {
		t.Fatal("a different hardware serial at the same fleet ID must be a mismatch")
	}
	if r.Device.Physical.Confidence != ConfidenceConflict {
		t.Fatalf("mismatch must drop confidence to conflict, got %s", r.Device.Physical.Confidence)
	}
}

func TestReconcileDisjointMACsMismatch(t *testing.T) {
	prev := Device{Physical: PhysicalIdentity{MACAddresses: []string{"88:dc:97:04:44:07"}, Confidence: ConfidenceVerified}}
	obs := Observed{MACAddresses: []string{"aa:bb:cc:dd:ee:ff"}}
	if r := Reconcile(prev, obs); !r.Mismatch {
		t.Fatal("fully disjoint MAC sets should be a mismatch")
	}
	// A shared MAC is fine.
	obs2 := Observed{MACAddresses: []string{"88:DC:97:04:44:07", "aa:bb:cc:dd:ee:ff"}}
	if r := Reconcile(prev, obs2); r.Mismatch {
		t.Fatalf("a shared MAC must not be a mismatch: %v", r.MismatchReasons)
	}
}

func TestReconcileFirstObservationFills(t *testing.T) {
	r := Reconcile(Device{FleetID: "a"}, Observed{FleetID: "a", Model: "ap-hk07"})
	if r.Device.Physical.Model != "ap-hk07" {
		t.Fatal("first observation should fill the model")
	}
	if r.Device.Physical.Confidence != ConfidenceUnknown {
		t.Fatalf("no hardware serial yet -> unknown confidence, got %s", r.Device.Physical.Confidence)
	}
}
