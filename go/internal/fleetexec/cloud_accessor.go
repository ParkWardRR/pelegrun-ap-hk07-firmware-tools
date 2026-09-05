package fleetexec

import (
	"context"
	"fmt"
	"strings"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/fleet"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/jess"
)

// CloudClient is the subset of jess.Cloud the CloudAccessor needs. *jess.Cloud
// satisfies it; tests pass a fake.
type CloudClient interface {
	Login(ctx context.Context, user, pass string) error
	SysInfo(ctx context.Context) (jess.SysInfo, error)
	UploadImage(ctx context.Context, filename string, data []byte) (size int, checksum string, err error)
	FwUpgrade(ctx context.Context) error
}

// CloudAccessor implements fleet.Accessor for the cloud/FIT HTTP families (no
// shell). Identity comes from /api/sys/sys_info; the flash is UploadImage (stage
// + device-side validate of the INACTIVE slot) followed by FwUpgrade
// (Upgrade_locally, which commits and reboots).
//
// ─────────────────────────────────────────────────────────────────────────────
// NOTES FOR THE NEXT AGENT (hardware pass):
//   - These families have NO shell, so the on-device `dump` path is unavailable.
//     Backup is therefore a REQUIRED injected hook (BackupFn): capture whatever
//     the API/out-of-band method allows (config export, image download) and
//     return its hash. A nil BackupFn errors — never a silent skip of the backup
//     safety gate.
//   - VERIFY the flash mapping on real hardware: this assumes UploadImage stages
//     to the inactive slot and returns the device's validated size+checksum, and
//     that FwUpgrade{Upgrade_locally} commits + reboots. If FwUpgrade instead
//     writes synchronously, the WriteInactive/Reboot split still holds (reboot is
//     just the commit), but confirm no active-slot overwrite occurs.
//   - Confirm sys_info.serial_number is the REAL device serial and is preserved
//     across the upgrade (P10 constraint). ExpectedFirmware lets Validate assert
//     the new version; set it from the plan.
//
// ─────────────────────────────────────────────────────────────────────────────
type CloudAccessor struct {
	Client           CloudClient
	User             string // default "admin"
	Pass             string // default "admin"
	ExpectedSerial   string // asserted at preflight + preserved at validate
	ExpectedFirmware string // optional: asserted at validate

	Image     []byte // firmware image bytes to stage
	ImageName string // filename for the upload

	// Required: cloud has no shell, so backup is supplied out-of-band.
	BackupFn func(ctx context.Context) (string, error)
	// Optional model-appropriate post-change health check.
	HealthFn func(ctx context.Context) (bool, error)

	stagedSize     int
	stagedChecksum string
}

var _ fleet.Accessor = (*CloudAccessor)(nil)

func (a *CloudAccessor) user() string {
	if a.User == "" {
		return "admin"
	}
	return a.User
}

func (a *CloudAccessor) pass() string {
	if a.Pass == "" {
		return "admin"
	}
	return a.Pass
}

func (a *CloudAccessor) login(ctx context.Context) error {
	if a.Client == nil {
		return fmt.Errorf("no cloud client configured")
	}
	return a.Client.Login(ctx, a.user(), a.pass())
}

// Preflight logs in and confirms the device identity.
func (a *CloudAccessor) Preflight(ctx context.Context) error {
	if err := a.login(ctx); err != nil {
		return fmt.Errorf("preflight login: %w", err)
	}
	si, err := a.Client.SysInfo(ctx)
	if err != nil {
		return fmt.Errorf("preflight sys_info: %w", err)
	}
	if a.ExpectedSerial != "" && !strings.EqualFold(a.ExpectedSerial, si.Serial) {
		return fmt.Errorf("preflight: expected serial %q, device reports %q", a.ExpectedSerial, si.Serial)
	}
	return nil
}

// Backup runs the required out-of-band backup hook (cloud has no shell dump).
func (a *CloudAccessor) Backup(ctx context.Context) (string, error) {
	if a.BackupFn == nil {
		return "", fmt.Errorf("Backup: cloud firmware has no shell; supply BackupFn for an out-of-band backup (config/image + hash) — the backup gate must not be skipped")
	}
	return a.BackupFn(ctx)
}

// WriteInactive stages the image; the device validates it and reports size+sum.
func (a *CloudAccessor) WriteInactive(ctx context.Context) error {
	if len(a.Image) == 0 {
		return fmt.Errorf("WriteInactive: no image bytes configured")
	}
	if err := a.login(ctx); err != nil {
		return fmt.Errorf("write login: %w", err)
	}
	size, sum, err := a.Client.UploadImage(ctx, a.imageName(), a.Image)
	if err != nil {
		return fmt.Errorf("stage image: %w", err)
	}
	a.stagedSize, a.stagedChecksum = size, sum
	return nil
}

// VerifyWrite confirms the device accepted and validated the staged image.
func (a *CloudAccessor) VerifyWrite(ctx context.Context) error {
	if a.stagedSize <= 0 || a.stagedChecksum == "" {
		return fmt.Errorf("VerifyWrite: device did not validate a staged image (size=%d checksum=%q)", a.stagedSize, a.stagedChecksum)
	}
	return nil
}

// Reboot commits the staged upgrade (Upgrade_locally) and reboots. The HTTP call
// often fails as the device drops — that transport error is expected, so it is
// not surfaced; validation reconnects.
func (a *CloudAccessor) Reboot(ctx context.Context) error {
	if a.Client == nil {
		return fmt.Errorf("reboot: no cloud client configured")
	}
	_ = a.Client.FwUpgrade(ctx)
	return nil
}

// Validate re-logs in, confirms the serial is preserved and (if set) the firmware
// updated, and runs the optional health check.
func (a *CloudAccessor) Validate(ctx context.Context) error {
	if err := a.login(ctx); err != nil {
		return fmt.Errorf("validate login: %w", err)
	}
	si, err := a.Client.SysInfo(ctx)
	if err != nil {
		return fmt.Errorf("validate sys_info: %w", err)
	}
	if a.ExpectedSerial != "" && !strings.EqualFold(a.ExpectedSerial, si.Serial) {
		return fmt.Errorf("validate: serial changed — expected %q, device reports %q", a.ExpectedSerial, si.Serial)
	}
	if a.ExpectedFirmware != "" && !strings.EqualFold(a.ExpectedFirmware, si.Firmware) {
		return fmt.Errorf("validate: expected firmware %q, device reports %q", a.ExpectedFirmware, si.Firmware)
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

func (a *CloudAccessor) imageName() string {
	if a.ImageName == "" {
		return "firmware.bin"
	}
	return a.ImageName
}
