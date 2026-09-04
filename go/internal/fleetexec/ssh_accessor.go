// Package fleetexec wires the pure fleet.Accessor contract to real device I/O.
// It composes swallow's existing read-only plans (hood env, dump capture) over an
// injected command Runner — jess.SSH.Run matches the Runner signature — while the
// family-specific DESTRUCTIVE steps (writing the inactive slot, verifying the
// written image, pointing the boot slot) are REQUIRED injected hooks. That split
// is deliberate: the safe, composable steps are implemented and unit-tested here;
// the dangerous, hardware- and firmware-family-specific steps are supplied by the
// caller and validated on real hardware. An unset destructive hook is an error,
// never a silent no-op.
//
// ─────────────────────────────────────────────────────────────────────────────
// NOTES FOR THE NEXT AGENT (hardware pass):
//   - Runner: pass jess.SSH{Host,User,Pass}.Run for the EWS/LuCI SSH:8822 path.
//     For the cloud/FIT families the identity re-read and flash are HTTP, not
//     shell — implement a sibling accessor (CloudAccessor) using jess.Cloud
//     (Login + sys_info gives serial/firmware/mac; UploadImage + FwUpgrade flash).
//   - Preflight/Validate confirm the u-boot env is COMPLETE (hood) and, if
//     ExpectedSerial is set, that it appears in the printenv output. VERIFY where
//     the real serial actually lives on your firmware — it may be an env var
//     (serial#, sn), in ART (mtd11), or only via the cloud API. Adjust
//     identityConfirmed() accordingly; do not weaken it to "always true".
//   - Backup runs the on-device dump plan (dd/nanddump to DumpDest) and returns
//     the SHA-256 of the on-device SHA256SUMS as the bundle id. Pulling the .bin
//     artifacts to the host (scp/sftp) and dump.Manifest.Verify is a follow-up
//     step this accessor does NOT do yet — add it.
//   - WriteInactiveFn MUST write the INACTIVE slot only (see internal/flash for
//     which slot that is) and MUST NOT touch ART. PointBootFn should append-only
//     set active_fw via hood (never a full env rewrite). VerifyWriteFn must prove
//     the written image (hash/length) before reboot.
//
// ─────────────────────────────────────────────────────────────────────────────
package fleetexec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/dump"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/fleet"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
)

// Runner executes one shell command on the device and returns its combined
// output. jess.SSH.Run satisfies this exactly.
type Runner func(ctx context.Context, cmd string) (string, error)

// SSHAccessor implements fleet.Accessor over a command Runner (EWS/LuCI SSH path).
type SSHAccessor struct {
	Run            Runner // required
	DumpDest       string // device-local dump dir (tmpfs/USB); default /tmp/swallow-dump
	NAND           bool   // use nanddump for the capture
	ExpectedSerial string // optional identity assertion checked at preflight/validate
	RebootCmd      string // default "reboot"

	// Required destructive hooks (family-specific). Nil => the step errors.
	WriteInactiveFn func(ctx context.Context) error // write the INACTIVE slot only
	VerifyWriteFn   func(ctx context.Context) error // prove the written image
	// Optional: append-only set active_fw to the new slot before reboot.
	PointBootFn func(ctx context.Context) error
	// Optional: model-appropriate post-change health check (beyond ping).
	HealthFn func(ctx context.Context) (bool, error)
}

// assert SSHAccessor satisfies the pure contract.
var _ fleet.Accessor = (*SSHAccessor)(nil)

func (a *SSHAccessor) dest() string {
	if a.DumpDest == "" {
		return "/tmp/swallow-dump"
	}
	return strings.TrimRight(a.DumpDest, "/")
}

// Preflight confirms the env is complete and (optionally) the expected identity.
func (a *SSHAccessor) Preflight(ctx context.Context) error {
	if a.Run == nil {
		return fmt.Errorf("preflight: no command Runner configured")
	}
	return a.identityConfirmed(ctx, "preflight")
}

// Backup captures the full flash on-device and returns the SHA-256 of the
// on-device SHA256SUMS as the bundle id. It refuses if a recovery-critical
// partition was not captured.
func (a *SSHAccessor) Backup(ctx context.Context) (string, error) {
	if a.Run == nil {
		return "", fmt.Errorf("backup: no command Runner configured")
	}
	mtd, err := a.Run(ctx, "cat /proc/mtd")
	if err != nil {
		return "", fmt.Errorf("read /proc/mtd: %w", err)
	}
	parts, err := dump.ParseProcMTD(mtd)
	if err != nil {
		return "", err
	}
	steps, err := dump.Plan(parts, a.dest(), dump.PlanOptions{NAND: a.NAND})
	if err != nil {
		return "", err
	}
	if _, err := a.Run(ctx, "mkdir -p "+a.dest()); err != nil {
		return "", fmt.Errorf("prepare dump dir: %w", err)
	}
	for _, s := range steps {
		if _, err := a.Run(ctx, s.Command); err != nil {
			return "", fmt.Errorf("dump step %q: %w", s.Desc, err)
		}
	}
	sums, err := a.Run(ctx, "cat "+a.dest()+"/SHA256SUMS")
	if err != nil {
		return "", fmt.Errorf("read SHA256SUMS: %w", err)
	}
	captured := parseSums(sums)
	if miss := dump.MissingCritical(parts, captured); len(miss) > 0 {
		return "", fmt.Errorf("backup missing recovery-critical artifacts: %v", miss)
	}
	sum := sha256.Sum256([]byte(sums))
	return hex.EncodeToString(sum[:]), nil
}

// WriteInactive runs the required family-specific inactive-slot write.
func (a *SSHAccessor) WriteInactive(ctx context.Context) error {
	if a.WriteInactiveFn == nil {
		return fmt.Errorf("WriteInactive: no write hook configured (family-specific; must write the INACTIVE slot only)")
	}
	return a.WriteInactiveFn(ctx)
}

// VerifyWrite runs the required proof of the written image.
func (a *SSHAccessor) VerifyWrite(ctx context.Context) error {
	if a.VerifyWriteFn == nil {
		return fmt.Errorf("VerifyWrite: no verify hook configured (must prove the written image before reboot)")
	}
	return a.VerifyWriteFn(ctx)
}

// Reboot optionally points the boot slot (append-only) then reboots.
func (a *SSHAccessor) Reboot(ctx context.Context) error {
	if a.PointBootFn != nil {
		if err := a.PointBootFn(ctx); err != nil {
			return fmt.Errorf("point boot slot: %w", err)
		}
	}
	if a.Run == nil {
		return fmt.Errorf("reboot: no command Runner configured")
	}
	cmd := a.RebootCmd
	if cmd == "" {
		cmd = "reboot"
	}
	// A reboot typically drops the connection; a non-nil error here is expected
	// and the executor moves to validation which reconnects. We surface only a
	// setup error, not the dropped-session error, by ignoring the run error.
	_, _ = a.Run(ctx, cmd)
	return nil
}

// Validate re-reads identity/env after reboot and runs the optional health check.
func (a *SSHAccessor) Validate(ctx context.Context) error {
	if a.Run == nil {
		return fmt.Errorf("validate: no command Runner configured")
	}
	if err := a.identityConfirmed(ctx, "validate"); err != nil {
		return err
	}
	if a.HealthFn != nil {
		ok, err := a.HealthFn(ctx)
		if err != nil {
			return fmt.Errorf("health check: %w", err)
		}
		if !ok {
			return fmt.Errorf("post-change health check did not pass")
		}
	}
	return nil
}

// identityConfirmed reads the env, requires it complete, and (if configured)
// checks the expected serial appears in the printenv output.
func (a *SSHAccessor) identityConfirmed(ctx context.Context, phase string) error {
	out, err := a.Run(ctx, "fw_printenv")
	if err != nil {
		return fmt.Errorf("%s: read env: %w", phase, err)
	}
	env := hood.ParsePrintenv(out)
	if !env.IsComplete() {
		return fmt.Errorf("%s: env incomplete, missing %v", phase, env.Missing())
	}
	if a.ExpectedSerial != "" && !strings.Contains(strings.ToLower(out), strings.ToLower(a.ExpectedSerial)) {
		return fmt.Errorf("%s: expected serial %q not present in device env", phase, a.ExpectedSerial)
	}
	return nil
}

// parseSums turns `sha256sum` output ("<hex>  <name>") into a set of captured
// artifact basenames.
func parseSums(s string) map[string]bool {
	set := map[string]bool{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[len(fields)-1]
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		set[name] = true
	}
	return set
}
