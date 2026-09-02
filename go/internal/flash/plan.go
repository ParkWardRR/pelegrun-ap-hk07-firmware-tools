package flash

import (
	"fmt"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/mews"
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

func countCritical() int {
	n := 0
	for _, a := range mews.Plan() {
		if a.Critical {
			n++
		}
	}
	return n
}
