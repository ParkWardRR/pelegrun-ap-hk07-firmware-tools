package fleetexec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/fleet"
	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/jess"
)

// fakeCloud implements CloudClient with scriptable behavior.
type fakeCloud struct {
	serial      string
	firmware    string
	afterSerial string // serial reported after reboot (defaults to serial)
	afterFW     string // firmware reported after reboot
	uploadErr   error
	rebooted    bool
	calls       []string
}

func (f *fakeCloud) Login(ctx context.Context, u, p string) error {
	f.calls = append(f.calls, "login")
	return nil
}
func (f *fakeCloud) SysInfo(ctx context.Context) (jess.SysInfo, error) {
	f.calls = append(f.calls, "sysinfo")
	s, fw := f.serial, f.firmware
	if f.rebooted {
		if f.afterSerial != "" {
			s = f.afterSerial
		}
		if f.afterFW != "" {
			fw = f.afterFW
		}
	}
	return jess.SysInfo{Serial: s, Firmware: fw}, nil
}
func (f *fakeCloud) UploadImage(ctx context.Context, name string, data []byte) (int, string, error) {
	f.calls = append(f.calls, "upload")
	if f.uploadErr != nil {
		return 0, "", f.uploadErr
	}
	return len(data), "sum-abc", nil
}
func (f *fakeCloud) FwUpgrade(ctx context.Context) error {
	f.calls = append(f.calls, "fwupgrade")
	f.rebooted = true
	return nil
}

func newCloudAccessor(fc *fakeCloud) *CloudAccessor {
	return &CloudAccessor{
		Client:           fc,
		ExpectedSerial:   "224203612",
		ExpectedFirmware: "2.0.0",
		Image:            []byte("firmware-bytes"),
		ImageName:        "ecw230v3.bin",
		BackupFn:         func(context.Context) (string, error) { return "backup-hash", nil },
	}
}

func TestCloudAccessorFullRun(t *testing.T) {
	fc := &fakeCloud{serial: "224203612", firmware: "1.8.112", afterFW: "2.0.0"}
	acc := newCloudAccessor(fc)
	j := fleet.NewJournal("op-1", "a", "0.4.0", time.Now())
	if err := fleet.Run(context.Background(), j, acc, fleet.Options{}); err != nil {
		t.Fatalf("cloud run: %v", err)
	}
	if j.State != fleet.StateCompleted {
		t.Fatalf("expected completed, got %s", j.State)
	}
	if !fc.rebooted {
		t.Fatal("FwUpgrade should have been called")
	}
}

func TestCloudBackupRequiresHook(t *testing.T) {
	acc := newCloudAccessor(&fakeCloud{serial: "224203612"})
	acc.BackupFn = nil
	if _, err := acc.Backup(context.Background()); err == nil {
		t.Fatal("cloud Backup must error without a BackupFn (backup gate not skippable)")
	}
}

func TestCloudPreflightSerialMismatch(t *testing.T) {
	acc := newCloudAccessor(&fakeCloud{serial: "999999999"})
	if err := acc.Preflight(context.Background()); err == nil {
		t.Fatal("preflight should reject a serial mismatch")
	}
}

func TestCloudValidateSerialChanged(t *testing.T) {
	// Serial changes after reboot -> P10 violation -> validate fails.
	fc := &fakeCloud{serial: "224203612", firmware: "1.8.112", afterSerial: "000000000", afterFW: "2.0.0"}
	acc := newCloudAccessor(fc)
	j := fleet.NewJournal("op-1", "a", "0.4.0", time.Now())
	err := fleet.Run(context.Background(), j, acc, fleet.Options{})
	if !errors.Is(err, fleet.ErrManualRecovery) || j.State != fleet.StatePausedManual {
		t.Fatalf("a changed serial at validate should require manual recovery, got state=%s err=%v", j.State, err)
	}
}

func TestCloudUploadFailureIsFailed(t *testing.T) {
	fc := &fakeCloud{serial: "224203612", firmware: "1.8.112", uploadErr: errors.New("device rejected")}
	acc := newCloudAccessor(fc)
	j := fleet.NewJournal("op-1", "a", "0.4.0", time.Now())
	err := fleet.Run(context.Background(), j, acc, fleet.Options{})
	if err == nil || j.State != fleet.StateFailed {
		t.Fatalf("upload failure should land in failed, got state=%s err=%v", j.State, err)
	}
}

func TestCloudVerifyWriteNeedsStaged(t *testing.T) {
	acc := newCloudAccessor(&fakeCloud{serial: "224203612"})
	// VerifyWrite before WriteInactive -> nothing staged -> error.
	if err := acc.VerifyWrite(context.Background()); err == nil {
		t.Fatal("VerifyWrite should fail when no image was staged")
	}
}
