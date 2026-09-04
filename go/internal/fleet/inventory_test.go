package fleet

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Devices) != 0 || f.SchemaVersion != SchemaVersion {
		t.Fatalf("expected empty fleet at schema %d, got %+v", SchemaVersion, f)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inv.json")
	in := &Fleet{Devices: []Device{{
		FleetID:  "a",
		Physical: PhysicalIdentity{Model: "ap-hk07", HardwareSerial: "SWLWX420001Q", Confidence: ConfidenceVerified},
		Software: SoftwareIdentity{ActiveSlot: "rootfs", InactiveSlot: "rootfs_1"},
		Access:   Access{ManagementAddress: "192.168.1.10", LastSeenAt: "2026-09-04T10:00:00Z"},
	}}}
	if err := in.Save(path); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Snapshot() != in.Snapshot() {
		t.Fatalf("snapshot changed across round trip:\n%s\n%s", in.Snapshot(), out.Snapshot())
	}
	if out.Get("a") == nil || out.Get("missing") != nil {
		t.Fatal("Get lookup wrong after round trip")
	}
}

func TestIsStale(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fresh := Device{Access: Access{LastSeenAt: now.Add(-30 * time.Second).Format(time.RFC3339)}}
	old := Device{Access: Access{LastSeenAt: now.Add(-2 * time.Hour).Format(time.RFC3339)}}
	never := Device{}

	if fresh.IsStale(now, time.Minute) {
		t.Fatal("30s-old reading should not be stale at 1m threshold")
	}
	if !old.IsStale(now, time.Minute) {
		t.Fatal("2h-old reading should be stale at 1m threshold")
	}
	if !never.IsStale(now, time.Minute) {
		t.Fatal("never-seen device should be treated as stale")
	}
	if never.IsStale(now, 0) {
		t.Fatal("zero threshold disables staleness")
	}
}
