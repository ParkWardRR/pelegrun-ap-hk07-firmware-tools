// Package flash implements the no-UART A/B firmware flash + verify (dev phase 4).
//
// It never executes I/O itself: it decides WHICH slot to write (always the
// inactive one — Constitution I), composes an ordered, gated plan, and the
// orchestrator runs the steps through a jess adapter. A failed flash therefore
// always leaves the active slot bootable.
package flash

import (
	"fmt"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/hood"
)

// The ap-hk07 dual-image slots.
const (
	SlotA = "rootfs"   // active_fw=0
	SlotB = "rootfs_1" // active_fw=1
)

// Slots returns the active and inactive rootfs slot names from the u-boot env.
// Defaults to A active when active_fw is absent/unparseable (matches the APPSBL
// compiled default), which is the safe assumption.
func Slots(env hood.Env) (active, inactive string) {
	switch env["active_fw"] {
	case "1":
		return SlotB, SlotA
	default:
		return SlotA, SlotB
	}
}

// NextActiveFW returns the active_fw value that selects the given slot.
func NextActiveFW(slot string) (string, error) {
	switch slot {
	case SlotA:
		return "0", nil
	case SlotB:
		return "1", nil
	default:
		return "", fmt.Errorf("unknown slot %q", slot)
	}
}

// FixedPartitionTarget is the single partition mainline OpenWrt's qualcommax
// target boots from, regardless of which slot `active_fw` currently selects.
//
// Unlike the FIT firmware's A/B pair, an OpenWrt kernel hardcodes root-mount
// to the DTS-labeled "rootfs" partition (SlotA) no matter which physical
// slot u-boot loaded it from — writing SlotB leaves an otherwise-valid image
// with a kernel that boots but can never find its own root filesystem. This
// was proven on real EWS377AP v3 hardware: see
// openwrt-ews377ap-v3/results-2026-09-06/STAGE3-SLOT0-INSTALL-RESULTS.md in
// this repo's history for the full trace, and the earlier attempt at
// openwrt-ews377ap-v3/results-2026-09-06/FLASH-STAGE2-RESULT.md that first
// found the wrong-slot hang.
//
// A direct consequence: writing this target when it's currently the
// *active* slot leaves no bootable A/B fallback during the write. Callers
// must not advertise A/B rollback for this path (see flash.PlanOpenWrt and
// adapter.CapFlashAB's doc comment).
const FixedPartitionTarget = SlotA
