package flash

import (
	"fmt"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/mews"
)

// Step is one ordered, human-readable action in a safe flash.
type Step struct {
	Desc string
	Note string
	Gate bool // a hard gate that must pass before proceeding
}

// ErrEnvIncomplete is returned when the env is too fragile to flash.
var ErrEnvIncomplete = hood.ErrIncomplete

// Plan builds the ordered safe-flash plan for a device. It refuses (error) if the
// current env is incomplete — flashing on a fragile env risks a slot that can't be
// pointed back. imageName is the (already product_id-patched) image to write.
func Plan(fam eyas.Family, env hood.Env, imageName string) ([]Step, error) {
	if !env.IsComplete() {
		return nil, fmt.Errorf("%w: missing %v — repair env before flashing", ErrEnvIncomplete, env.Missing())
	}
	active, inactive := Slots(env)
	how := "cloud upload.cgi → local_upgrade_image → fw_upgrade{Upgrade_locally}"
	if fam == eyas.EwsLuCI {
		how = "LuCI flashops (upload → step=2 confirm)"
	}
	return []Step{
		{Desc: "backup bundle (mtd7/8/11 + config + hashes)", Note: fmt.Sprintf("%d critical artifacts", countCritical()), Gate: true},
		{Desc: "verify env is complete", Note: "bootcmd/active_fw/app_part/rootfsname present", Gate: true},
		{Desc: "write the INACTIVE slot", Note: fmt.Sprintf("active=%s → write %s via %s", active, inactive, how), Gate: false},
		{Desc: "point boot at the new slot", Note: fmt.Sprintf("append-only: active_fw for %s", inactive), Gate: false},
		{Desc: "reboot and watch", Note: "OS up + on-network within timeout", Gate: false},
		{Desc: "verify identity", Note: "serial + MAC re-read match intent", Gate: true},
		{Desc: "rollback available", Note: fmt.Sprintf("reset-button hold reverts to %s", active), Gate: false},
	}, nil
}

// PlanOpenWrt builds the ordered safe-flash plan for installing mainline
// OpenWrt. It differs from Plan (the FIT A/B path) in two load-bearing ways:
//
//  1. It always targets flash.FixedPartitionTarget, never the computed
//     "inactive" slot — OpenWrt's root-mount requires it (see
//     FixedPartitionTarget's doc comment).
//  2. It does not claim A/B rollback. Writing the fixed target while it's
//     the active slot forfeits the live fallback; the plan says so and
//     names the real recovery route (byte-exact backup + UART/TFTP
//     recovery), rather than reusing the FIT path's "reset-button hold"
//     step, which would be actively false here.
//
// imageName should be a UBI factory image already confirmed (by `quarry
// verify-ubi`) to carry a board-matched FIT config node; Plan does not
// re-parse the image itself (see the "verify image" gate below, and
// adapter.CapFlashAB's doc comment for why this path can't declare that
// capability).
func PlanOpenWrt(env hood.Env, imageName string) ([]Step, error) {
	if !env.IsComplete() {
		return nil, fmt.Errorf("%w: missing %v — repair env before flashing", ErrEnvIncomplete, env.Missing())
	}
	target := FixedPartitionTarget
	return []Step{
		{Desc: "backup bundle (mtd7/8/11 + config + hashes)", Note: fmt.Sprintf("%d critical artifacts", countCritical()), Gate: true},
		{Desc: "verify env is complete", Note: "bootcmd/active_fw/app_part/rootfsname present", Gate: true},
		{Desc: "verify image", Note: fmt.Sprintf("%s: 'kernel' volume has /configurations/config@<board> (quarry verify-ubi)", imageName), Gate: true},
		{Desc: "verify ART is excluded", Note: "mtd11 (RF calibration + factory MAC) is never in this plan's write region", Gate: true},
		{Desc: "write the FIXED rootfs partition", Note: fmt.Sprintf("target=%s regardless of active_fw=%s — OpenWrt has no A/B choice here", target, env["active_fw"]), Gate: false},
		{Desc: "point boot at the fixed partition", Note: fmt.Sprintf("append-only: active_fw for %s", target), Gate: false},
		{Desc: "reboot and watch", Note: "OS up + on-network within timeout", Gate: false},
		{Desc: "verify identity", Note: "serial + MAC re-read match intent", Gate: true},
		{Desc: "no live A/B fallback", Note: "recovery route is byte-exact backup + UART/TFTP re-flash, NOT a reset-button hold — this write replaced the only slot OpenWrt boots from", Gate: false},
	}, nil
}

func countCritical() int {
	n := 0
	for _, a := range mews.Plan() {
		if a.Critical {
			n++
		}
	}
	return n
}
