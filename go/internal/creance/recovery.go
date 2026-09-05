// Package creance drives the UART console recovery (the long training line):
// the GATED "prove before persist" u-boot sequence. The pure decision logic lives
// here and is tested; the serial I/O is a thin adapter on top.
//
// The hazard this guards against (Constitution I): an erased env is safe (u-boot
// falls back to complete compiled defaults), but a valid-but-INCOMPLETE env is
// worse — u-boot trusts it and any missing boot var bricks the slot. So we never
// `env save` a candidate that isn't complete.
package creance

import (
	"fmt"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/hood"
)

// Cmd is one u-boot console command with why it runs.
type Cmd struct {
	Line string
	Why  string
	Gate bool // after this, apply Decide() before continuing
}

// Script is the ordered, gated recovery sequence for a stuck slot.
// It loads defaults into RAM, INSPECTS them, and only persists if Decide() passes.
func Script() []Cmd {
	return []Cmd{
		{"printenv", "capture the current (bad) env to the console log", false},
		{"version ; help env", "confirm this u-boot supports `env default`", false},
		{"env default -a", "load complete compiled defaults into RAM ONLY", false},
		{"printenv bootcmd active_fw app_part rootfsname", "INSPECT the candidate before writing", true},
		{"env save", "persist — ONLY if Decide() passed", false},
		{"env load ; printenv bootcmd active_fw", "prove persistence, don't assume", false},
		{"reset", "boot once (explicit boot cmd if understood)", false},
	}
}

// ParseUboot parses a u-boot `printenv` dump (key=value lines) into an env.
func ParseUboot(dump string) hood.Env { return hood.ParsePrintenv(dump) }

// Decide inspects the RAM-default env (after `env default -a`) and reports whether
// it is safe to `env save`, or a reason to STOP. This is the UART decision gate.
func Decide(ramDefaults hood.Env) (safe bool, reason string) {
	if len(ramDefaults) == 0 {
		return false, "empty/unreadable defaults — possibly a vendor-modified u-boot; capture output, don't guess"
	}
	if missing := ramDefaults.Missing(); len(missing) > 0 {
		return false, fmt.Sprintf("defaults missing %v — saving would recreate the failure; inspect the APPSBL default env + mtd map first", missing)
	}
	if bc := ramDefaults["bootcmd"]; bc == "" {
		return false, "no bootcmd in defaults"
	}
	return true, "defaults are complete and coherent — safe to env save"
}
