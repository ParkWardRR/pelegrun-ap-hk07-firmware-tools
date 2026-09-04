package fleet

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestJournalHappyPath(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	seq := []OpState{
		StatePreflightPassed, StateBackupRunning, StateBackupCaptured,
		StateWriteInProgress, StateWriteVerified, StateRebootPending,
		StateValidationRunning, StateCompleted,
	}
	at := refNow
	for _, s := range seq {
		at = at.Add(time.Minute)
		if err := j.Advance(s, at, ""); err != nil {
			t.Fatalf("advance to %s: %v", s, err)
		}
	}
	if j.State != StateCompleted || !IsTerminal(j.State) {
		t.Fatalf("final state should be terminal completed, got %s", j.State)
	}
	if len(j.Transitions) != len(seq)+1 { // +1 for the opening "" -> planned
		t.Fatalf("expected %d transitions, got %d", len(seq)+1, len(j.Transitions))
	}
}

func TestJournalIllegalTransition(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	// Cannot skip straight from planned to write.
	err := j.Advance(StateWriteInProgress, refNow, "")
	var ill ErrIllegalTransition
	if !errors.As(err, &ill) {
		t.Fatalf("skipping states should be illegal, got %v", err)
	}
	if j.State != StatePlanned {
		t.Fatal("an illegal transition must not change state")
	}
}

func TestJournalCanDivergeToFailOrPause(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	_ = j.Advance(StateBackupRunning, refNow, "")
	if err := j.Advance(StatePausedManual, refNow, "operator halt"); err != nil {
		t.Fatalf("any live state may pause, got %v", err)
	}
	// Terminal: no further transitions.
	if err := j.Advance(StateBackupCaptured, refNow, ""); err == nil {
		t.Fatal("a paused (terminal) journal must not advance")
	}
}

func TestFailFromMidFlight(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	if !CanTransition(StatePreflightPassed, StateFailed) {
		t.Fatal("failure must be reachable from a live state")
	}
	if err := j.Advance(StateFailed, refNow, "adapter lost"); err != nil {
		t.Fatalf("fail transition: %v", err)
	}
	if !IsTerminal(StateFailed) {
		t.Fatal("failed is terminal")
	}
}

func TestAllowedNextTerminalEmpty(t *testing.T) {
	if AllowedNext(StateCompleted) != nil {
		t.Fatal("completed has no next states")
	}
	if len(AllowedNext(StateValidationRunning)) != 4 { // canary, completed, failed, paused
		t.Fatalf("validation-running next = %v", AllowedNext(StateValidationRunning))
	}
}

func TestJournalRefAndRoundTrip(t *testing.T) {
	j := NewJournal("op-1", "a", "0.4.0", refNow)
	_ = j.Advance(StatePreflightPassed, refNow, "")
	ref := j.Ref()
	if ref.OperationID != "op-1" || ref.State != StatePreflightPassed || ref.JournalSHA256 == "" {
		t.Fatalf("bad ref: %+v", ref)
	}
	path := filepath.Join(t.TempDir(), "j.json")
	if err := j.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ref().JournalSHA256 != ref.JournalSHA256 {
		t.Fatal("journal digest changed across round trip")
	}
}
