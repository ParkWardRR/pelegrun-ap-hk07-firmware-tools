package flash

import (
	"errors"
	"testing"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
)

func TestSlotsAlwaysInactive(t *testing.T) {
	a, i := Slots(hood.ParsePrintenv("active_fw=0\n"))
	if a != SlotA || i != SlotB {
		t.Fatalf("fw=0: active=%s inactive=%s", a, i)
	}
	a, i = Slots(hood.ParsePrintenv("active_fw=1\n"))
	if a != SlotB || i != SlotA {
		t.Fatalf("fw=1: active=%s inactive=%s", a, i)
	}
	// absent -> default A active (safe)
	a, _ = Slots(hood.ParsePrintenv("bootcmd=bootipq\n"))
	if a != SlotA {
		t.Fatalf("absent active_fw should default to A, got %s", a)
	}
}

func TestNextActiveFW(t *testing.T) {
	if v, _ := NextActiveFW(SlotA); v != "0" {
		t.Fatal("A->0")
	}
	if v, _ := NextActiveFW(SlotB); v != "1" {
		t.Fatal("B->1")
	}
	if _, err := NextActiveFW("bogus"); err == nil {
		t.Fatal("bogus slot should error")
	}
}

const completeEnv = "bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\n"

func TestPlanRefusesIncompleteEnv(t *testing.T) {
	_, err := Plan(eyas.Cloud, hood.ParsePrintenv("ethaddr=x\n"), "img.bin")
	if !errors.Is(err, ErrEnvIncomplete) {
		t.Fatalf("expected env-incomplete refusal, got %v", err)
	}
}

func TestPlanWritesInactiveSlot(t *testing.T) {
	steps, err := Plan(eyas.Cloud, hood.ParsePrintenv(completeEnv), "ecw230v3-282.bin")
	if err != nil {
		t.Fatal(err)
	}
	// must have a backup gate first and a write-inactive step naming rootfs_1
	if !steps[0].Gate {
		t.Fatal("first step must be a gate (backup)")
	}
	foundInactive := false
	for _, s := range steps {
		if s.Desc == "write the INACTIVE slot" && contains(s.Note, "write rootfs_1") {
			foundInactive = true
		}
	}
	if !foundInactive {
		t.Fatal("plan must write the inactive slot (rootfs_1) when active is rootfs")
	}
}

func TestPlanCloudVsLuCI(t *testing.T) {
	cloud, _ := Plan(eyas.Cloud, hood.ParsePrintenv(completeEnv), "x")
	luci, _ := Plan(eyas.EwsLuCI, hood.ParsePrintenv(completeEnv), "x")
	if !contains(noteOf(cloud, "write the INACTIVE slot"), "fw_upgrade") {
		t.Fatal("cloud plan should use fw_upgrade")
	}
	if !contains(noteOf(luci, "write the INACTIVE slot"), "flashops") {
		t.Fatal("luci plan should use flashops")
	}
}

func noteOf(steps []Step, desc string) string {
	for _, s := range steps {
		if s.Desc == desc {
			return s.Note
		}
	}
	return ""
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
