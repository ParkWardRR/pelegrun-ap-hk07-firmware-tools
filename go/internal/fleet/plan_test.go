package fleet

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func twoDeviceFleet() *Fleet {
	a := verifiedDevice() // "a", eligible
	b := verifiedDevice()
	b.FleetID = "b"
	b.Physical.HardwareSerial = "SWLWX420002N"
	c := verifiedDevice()
	c.FleetID = "c"
	c.Physical.Confidence = ConfidenceProbable // blocked (low confidence)
	c.Physical.HardwareSerial = "SWLWX420003H"
	return &Fleet{SchemaVersion: SchemaVersion, Devices: []Device{a, b, c}}
}

var tgt = Target{ImageName: "ecw230v3-282.bin", ImageSHA256: "deadbeef", FirmwareFamily: "cloud"}

func TestBuildPlanEligibilityAndCohorts(t *testing.T) {
	pl := BuildPlan(twoDeviceFleet(), basePolicy(), tgt, "0.4.0", refNow)
	if pl.EligibleCount != 2 || pl.BlockedCount != 1 {
		t.Fatalf("expected 2 eligible / 1 blocked, got %d/%d", pl.EligibleCount, pl.BlockedCount)
	}
	// Entries are sorted by fleet ID.
	if pl.Entries[0].FleetID != "a" || pl.Entries[2].FleetID != "c" {
		t.Fatal("entries must be sorted by fleet ID")
	}
	if pl.Entries[2].Eligible || pl.Entries[2].Cohort != CohortBlocked {
		t.Fatal("device c must be blocked with no cohort")
	}
	// Every eligible device is written to the inactive slot only.
	for _, e := range pl.Entries {
		if e.Eligible && e.InactiveSlot != "rootfs_1" {
			t.Fatalf("%s eligible but inactive slot = %q", e.FleetID, e.InactiveSlot)
		}
	}
}

func TestBuildPlanDeterministicHash(t *testing.T) {
	p1 := BuildPlan(twoDeviceFleet(), basePolicy(), tgt, "0.4.0", refNow)
	p2 := BuildPlan(twoDeviceFleet(), basePolicy(), tgt, "0.4.0", refNow)
	if p1.PlanHash == "" || p1.PlanHash != p2.PlanHash {
		t.Fatalf("identical inputs must yield identical plan hash: %q vs %q", p1.PlanHash, p2.PlanHash)
	}
}

func TestApplyGuardAcceptsCleanPlan(t *testing.T) {
	f := twoDeviceFleet()
	pl := BuildPlan(f, basePolicy(), tgt, "0.4.0", refNow)
	if err := ApplyGuard(pl, f, basePolicy(), tgt, refNow, 10*time.Minute); err != nil {
		t.Fatalf("clean plan should apply, got %v", err)
	}
}

func TestApplyGuardRefusals(t *testing.T) {
	build := func() (*Plan, *Fleet) {
		f := twoDeviceFleet()
		return BuildPlan(f, basePolicy(), tgt, "0.4.0", refNow), f
	}

	t.Run("tampered plan", func(t *testing.T) {
		pl, f := build()
		pl.Entries[0].Eligible = true
		pl.Entries[0].InactiveSlot = "rootfs" // sneak an edit after hashing
		if err := ApplyGuard(pl, f, basePolicy(), tgt, refNow, time.Hour); !errors.Is(err, ErrPlanTampered) {
			t.Fatalf("want ErrPlanTampered, got %v", err)
		}
	})

	t.Run("changed policy", func(t *testing.T) {
		pl, f := build()
		p2 := basePolicy()
		p2.RequireBackup = false
		if err := ApplyGuard(pl, f, p2, tgt, refNow, time.Hour); !errors.Is(err, ErrPolicyChanged) {
			t.Fatalf("want ErrPolicyChanged, got %v", err)
		}
	})

	t.Run("changed target", func(t *testing.T) {
		pl, f := build()
		t2 := tgt
		t2.ImageSHA256 = "cafef00d"
		if err := ApplyGuard(pl, f, basePolicy(), t2, refNow, time.Hour); !errors.Is(err, ErrTargetChanged) {
			t.Fatalf("want ErrTargetChanged, got %v", err)
		}
	})

	t.Run("identity mismatch", func(t *testing.T) {
		pl, f := build()
		f.Get("a").Physical.HardwareSerial = "SWAPPED00009" // device swapped after planning
		if err := ApplyGuard(pl, f, basePolicy(), tgt, refNow, time.Hour); !errors.Is(err, ErrIdentityMismatch) {
			t.Fatalf("want ErrIdentityMismatch, got %v", err)
		}
	})

	t.Run("device removed", func(t *testing.T) {
		pl, f := build()
		f.Devices = f.Devices[1:] // drop "a"
		if err := ApplyGuard(pl, f, basePolicy(), tgt, refNow, time.Hour); !errors.Is(err, ErrDeviceMissing) {
			t.Fatalf("want ErrDeviceMissing, got %v", err)
		}
	})

	t.Run("stale reading", func(t *testing.T) {
		pl, f := build()
		f.Get("a").Access.LastSeenAt = refNow.Add(-2 * time.Hour).Format(time.RFC3339)
		if err := ApplyGuard(pl, f, basePolicy(), tgt, refNow, 10*time.Minute); !errors.Is(err, ErrDiscoveryStale) {
			t.Fatalf("want ErrDiscoveryStale, got %v", err)
		}
	})

	t.Run("window closed", func(t *testing.T) {
		f := twoDeviceFleet()
		p := basePolicy()
		p.MaintenanceWindow = &Window{StartRFC3339: "2026-09-04T10:00:00Z", EndRFC3339: "2026-09-04T14:00:00Z"}
		pl := BuildPlan(f, p, tgt, "0.4.0", refNow)
		if err := ApplyGuard(pl, f, p, tgt, refNow.Add(6*time.Hour), time.Hour); !errors.Is(err, ErrWindowClosed) {
			t.Fatalf("want ErrWindowClosed, got %v", err)
		}
	})
}

func TestPlanSaveLoadRoundTrip(t *testing.T) {
	f := twoDeviceFleet()
	pl := BuildPlan(f, basePolicy(), tgt, "0.4.0", refNow)
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := pl.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	// A faithfully round-tripped plan still applies (hash intact).
	if err := ApplyGuard(got, f, basePolicy(), tgt, refNow, time.Hour); err != nil {
		t.Fatalf("round-tripped plan should still apply, got %v", err)
	}
}
