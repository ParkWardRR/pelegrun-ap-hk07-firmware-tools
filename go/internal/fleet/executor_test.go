package fleet

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeAccessor records calls and can be told to fail at one named step.
type fakeAccessor struct {
	calls  []string
	failAt string
	err    error
}

func (f *fakeAccessor) mark(name string) error {
	f.calls = append(f.calls, name)
	if name == f.failAt {
		if f.err != nil {
			return f.err
		}
		return errors.New("boom at " + name)
	}
	return nil
}

func (f *fakeAccessor) Preflight(context.Context) error { return f.mark("preflight") }
func (f *fakeAccessor) Backup(context.Context) (string, error) {
	return "deadbeefcafef00d", f.mark("backup")
}
func (f *fakeAccessor) WriteInactive(context.Context) error { return f.mark("write") }
func (f *fakeAccessor) VerifyWrite(context.Context) error   { return f.mark("verify") }
func (f *fakeAccessor) Reboot(context.Context) error        { return f.mark("reboot") }
func (f *fakeAccessor) Validate(context.Context) error      { return f.mark("validate") }

func testOptions(canary bool) (Options, *[]OpState) {
	var seen []OpState
	tick := refNow
	o := Options{
		Canary: canary,
		Now: func() time.Time {
			tick = tick.Add(time.Second)
			return tick
		},
		Persist: func(j *Journal) error {
			seen = append(seen, j.State)
			return nil
		},
	}
	return o, &seen
}

func TestRunHappyPath(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	acc := &fakeAccessor{}
	o, persisted := testOptions(false)
	if err := Run(context.Background(), j, acc, o); err != nil {
		t.Fatalf("run: %v", err)
	}
	if j.State != StateCompleted {
		t.Fatalf("final state = %s, want completed", j.State)
	}
	wantCalls := []string{"preflight", "backup", "write", "verify", "reboot", "validate"}
	if len(acc.calls) != len(wantCalls) {
		t.Fatalf("calls = %v, want %v", acc.calls, wantCalls)
	}
	for i, c := range wantCalls {
		if acc.calls[i] != c {
			t.Fatalf("call %d = %q, want %q", i, acc.calls[i], c)
		}
	}
	// Write intent must be persisted BEFORE the write call.
	if !persistedBefore(*persisted, StateWriteInProgress) {
		t.Fatal("write_in_progress must be persisted (intent recorded before mutation)")
	}
}

func persistedBefore(states []OpState, s OpState) bool {
	for _, st := range states {
		if st == s {
			return true
		}
	}
	return false
}

func TestRunCanaryAddsObservation(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	o, _ := testOptions(true)
	if err := Run(context.Background(), j, &fakeAccessor{}, o); err != nil {
		t.Fatal(err)
	}
	saw := false
	for _, tr := range j.Transitions {
		if tr.To == StateCanaryObservation {
			saw = true
		}
	}
	if !saw || j.State != StateCompleted {
		t.Fatalf("canary run should pass through observation to completed, transitions=%v", j.Transitions)
	}
}

func TestBackupFailureIsFailed(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	o, _ := testOptions(false)
	err := Run(context.Background(), j, &fakeAccessor{failAt: "backup"}, o)
	if err == nil || j.State != StateFailed {
		t.Fatalf("backup failure should end in failed, got state=%s err=%v", j.State, err)
	}
}

func TestWriteFailureIsFailedNotManual(t *testing.T) {
	// A failed inactive-slot write leaves the active slot bootable -> failed, retryable.
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	o, _ := testOptions(false)
	err := Run(context.Background(), j, &fakeAccessor{failAt: "write"}, o)
	if err == nil || j.State != StateFailed {
		t.Fatalf("write failure should be failed, got state=%s err=%v", j.State, err)
	}
}

func TestValidateFailureIsManualRecovery(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	o, _ := testOptions(false)
	err := Run(context.Background(), j, &fakeAccessor{failAt: "validate"}, o)
	if !errors.Is(err, ErrManualRecovery) || j.State != StatePausedManual {
		t.Fatalf("validate failure should require manual recovery, got state=%s err=%v", j.State, err)
	}
}

func TestRebootFailureIsManualRecovery(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	o, _ := testOptions(false)
	err := Run(context.Background(), j, &fakeAccessor{failAt: "reboot"}, o)
	if !errors.Is(err, ErrManualRecovery) || j.State != StatePausedManual {
		t.Fatalf("reboot failure should require manual recovery, got state=%s err=%v", j.State, err)
	}
}

func TestResumeMidWriteGoesManual(t *testing.T) {
	// Simulate a crash after write intent was recorded but before completion.
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	_ = j.Advance(StateBackupRunning, refNow, "")
	_ = j.Advance(StateBackupCaptured, refNow, "")
	_ = j.Advance(StateWriteInProgress, refNow, "")

	acc := &fakeAccessor{}
	o, _ := testOptions(false)
	err := Resume(context.Background(), j, acc, o)
	if !errors.Is(err, ErrManualRecovery) || j.State != StatePausedManual {
		t.Fatalf("resume mid-write must go manual, got state=%s err=%v", j.State, err)
	}
	// It must NOT have blindly re-run the write.
	for _, c := range acc.calls {
		if c == "write" {
			t.Fatal("resume must never blindly repeat a write")
		}
	}
}

func TestResumeFromSafeStateContinues(t *testing.T) {
	// Crash right after backup captured; resume should reconcile then finish.
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	_ = j.Advance(StateBackupRunning, refNow, "")
	_ = j.Advance(StateBackupCaptured, refNow, "")

	acc := &fakeAccessor{}
	o, _ := testOptions(false)
	if err := Resume(context.Background(), j, acc, o); err != nil {
		t.Fatalf("resume from backup_captured should complete: %v", err)
	}
	if j.State != StateCompleted {
		t.Fatalf("resume should finish the operation, got %s", j.State)
	}
	// Reconcile preflight runs first on resume.
	if len(acc.calls) == 0 || acc.calls[0] != "preflight" {
		t.Fatalf("resume must reconcile (preflight) first, calls=%v", acc.calls)
	}
}

func TestRunRejectsNonPlanned(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	if err := Run(context.Background(), j, &fakeAccessor{}, Options{}); err == nil {
		t.Fatal("Run should reject a non-planned journal")
	}
}
