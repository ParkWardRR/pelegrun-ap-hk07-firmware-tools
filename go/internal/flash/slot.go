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
