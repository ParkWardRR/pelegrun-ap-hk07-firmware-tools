package fleetexec

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/fleet"
)

const procMTD = `dev:    size   erasesize  name
mtd7: 00080000 00020000 "0:APPSBLENV"
mtd8: 00140000 00020000 "0:APPSBL"
mtd11: 00080000 00020000 "0:ART"
`

const completeEnv = "bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nserial#=SWLWX420001Q\n"

// sums covering the three recovery-critical artifacts + the layout file.
const sums = `aaaa  mtd7-0-appsblenv.bin
bbbb  mtd8-0-appsbl.bin
cccc  mtd11-0-art.bin
dddd  proc-mtd.txt
`

func okRunner(procmtd, env, sha string) Runner {
	return func(ctx context.Context, cmd string) (string, error) {
		switch {
		case strings.Contains(cmd, "SHA256SUMS"):
			return sha, nil
		case strings.HasPrefix(cmd, "cat /proc/mtd"):
			return procmtd, nil
		case strings.HasPrefix(cmd, "fw_printenv"):
			return env, nil
		default:
			return "", nil
		}
	}
}

func newAccessor(r Runner) *SSHAccessor {
	return &SSHAccessor{
		Run:             r,
		ExpectedSerial:  "SWLWX420001Q",
		WriteInactiveFn: func(context.Context) error { return nil },
		VerifyWriteFn:   func(context.Context) error { return nil },
	}
}

func TestAccessorDrivesExecutorToCompletion(t *testing.T) {
	acc := newAccessor(okRunner(procMTD, completeEnv, sums))
	j := fleet.NewJournal("op-1", "a", "0.4.0", time.Now())
	if err := fleet.Run(context.Background(), j, acc, fleet.Options{}); err != nil {
		t.Fatalf("executor run through SSHAccessor: %v", err)
	}
	if j.State != fleet.StateCompleted {
		t.Fatalf("expected completed, got %s", j.State)
	}
}

func TestBackupRefusesMissingCritical(t *testing.T) {
	// SHA256SUMS without the ART artifact -> backup must refuse.
	partial := "aaaa  mtd7-0-appsblenv.bin\nbbbb  mtd8-0-appsbl.bin\n"
	acc := newAccessor(okRunner(procMTD, completeEnv, partial))
	if _, err := acc.Backup(context.Background()); err == nil {
		t.Fatal("backup should refuse when a recovery-critical artifact is missing")
	}
}

func TestBackupReturnsStableBundleID(t *testing.T) {
	acc := newAccessor(okRunner(procMTD, completeEnv, sums))
	id1, err := acc.Backup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := acc.Backup(context.Background())
	if id1 == "" || id1 != id2 {
		t.Fatalf("bundle id should be a stable hash of SHA256SUMS: %q vs %q", id1, id2)
	}
}

func TestPreflightRejectsIncompleteEnv(t *testing.T) {
	acc := newAccessor(okRunner(procMTD, "ethaddr=x\n", sums))
	if err := acc.Preflight(context.Background()); err == nil {
		t.Fatal("preflight should reject an incomplete env")
	}
}

func TestPreflightRejectsWrongSerial(t *testing.T) {
	env := "bootcmd=b\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nserial#=OTHER0000009\n"
	acc := newAccessor(okRunner(procMTD, env, sums))
	if err := acc.Preflight(context.Background()); err == nil {
		t.Fatal("preflight should reject when the expected serial is absent")
	}
}

func TestUnconfiguredWriteHookErrors(t *testing.T) {
	acc := &SSHAccessor{Run: okRunner(procMTD, completeEnv, sums)}
	if err := acc.WriteInactive(context.Background()); err == nil {
		t.Fatal("WriteInactive must error when no write hook is configured (never a silent no-op)")
	}
	if err := acc.VerifyWrite(context.Background()); err == nil {
		t.Fatal("VerifyWrite must error when no verify hook is configured")
	}
}

func TestWriteFailurePropagatesToExecutor(t *testing.T) {
	acc := newAccessor(okRunner(procMTD, completeEnv, sums))
	acc.WriteInactiveFn = func(context.Context) error { return context.DeadlineExceeded }
	j := fleet.NewJournal("op-1", "a", "0.4.0", time.Now())
	err := fleet.Run(context.Background(), j, acc, fleet.Options{})
	if err == nil || j.State != fleet.StateFailed {
		t.Fatalf("a write-hook failure should land the journal in failed, got state=%s err=%v", j.State, err)
	}
}

func TestPointBootRunsBeforeReboot(t *testing.T) {
	order := []string{}
	acc := newAccessor(func(ctx context.Context, cmd string) (string, error) {
		if strings.HasPrefix(cmd, "reboot") {
			order = append(order, "reboot")
		}
		return okRunner(procMTD, completeEnv, sums)(ctx, cmd)
	})
	acc.PointBootFn = func(context.Context) error { order = append(order, "pointboot"); return nil }
	if err := acc.Reboot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "pointboot" || order[1] != "reboot" {
		t.Fatalf("point-boot must run before reboot, got %v", order)
	}
}
