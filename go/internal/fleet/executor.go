package fleet

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Accessor performs the actual device I/O for one operation, in the order the
// executor calls it. Every method is injected so the executor stays pure and
// testable: the real implementation drives a jess adapter; tests pass a fake.
// Backup returns the captured bundle's SHA-256 for the journal.
type Accessor interface {
	Preflight(context.Context) error        // revalidate identity/eligibility on the device
	Backup(context.Context) (string, error) // capture the mews evidence bundle
	WriteInactive(context.Context) error    // write the INACTIVE slot (never the active one)
	VerifyWrite(context.Context) error      // verify the written image
	Reboot(context.Context) error           // point boot at the new slot and reboot
	Validate(context.Context) error         // post-change identity/health checks
}

// Persister durably stores a journal. The executor calls it after every material
// state transition — and crucially before every irreversible action — so an
// interrupted run can be reconciled instead of blindly replayed.
type Persister func(*Journal) error

// Options tune a run. Now and Persist default to time.Now().UTC() and a no-op.
type Options struct {
	Now     func() time.Time
	Persist Persister
	Canary  bool // this device is in the canary cohort — add an observation state
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now().UTC()
}

func (o Options) persist(j *Journal) error {
	if o.Persist != nil {
		return o.Persist(j)
	}
	return nil
}

// ErrManualRecovery means the operation stopped in paused_manual_recovery: an
// action failed after a mutation began, or a resume hit a state that must not be
// auto-retried. A human (or a fresh plan) must decide the next step.
var ErrManualRecovery = errors.New("operation requires manual recovery")

// Run drives a fresh journal (state planned) to a terminal state. It persists the
// journal after every transition; a mutation's intent is always persisted before
// the mutation runs (the *_in_progress / *_pending states), so a crash mid-write
// is visible on disk. On a failed read-only step it records failed; on a failure
// after a mutation began it records paused_manual_recovery.
func Run(ctx context.Context, j *Journal, acc Accessor, o Options) error {
	if j.State != StatePlanned {
		return fmt.Errorf("Run expects a planned journal, got %s (use Resume)", j.State)
	}
	return forward(ctx, j, acc, o)
}

// Resume continues a persisted, non-terminal journal. It refuses to auto-repeat a
// mutation that may be half-applied: from inactive_slot_write_in_progress or
// reboot_pending it goes straight to paused_manual_recovery. From any other live
// state it re-checks device identity (Preflight) before continuing forward.
func Resume(ctx context.Context, j *Journal, acc Accessor, o Options) error {
	if IsTerminal(j.State) {
		if j.State == StateFailed {
			return fmt.Errorf("journal already failed")
		}
		return nil
	}
	switch j.State {
	case StateWriteInProgress, StateRebootPending:
		return fail(j, StatePausedManual, "resumed mid-mutation; not auto-retrying "+string(j.State), o, ErrManualRecovery)
	}
	if err := acc.Preflight(ctx); err != nil {
		return fail(j, StatePausedManual, "reconcile Preflight failed on resume: "+err.Error(), o, ErrManualRecovery)
	}
	return forward(ctx, j, acc, o)
}

// forward runs the pipeline one state at a time until a terminal state.
func forward(ctx context.Context, j *Journal, acc Accessor, o Options) error {
	for !IsTerminal(j.State) {
		if err := doNext(ctx, j, acc, o); err != nil {
			return err
		}
	}
	return nil
}

// doNext performs the single action/transition appropriate to the current state.
// States ending in _running / _in_progress / _pending are "intent recorded,
// action in flight"; their paired done-state records success.
func doNext(ctx context.Context, j *Journal, acc Accessor, o Options) error {
	switch j.State {
	case StatePlanned:
		if err := acc.Preflight(ctx); err != nil {
			return fail(j, StateFailed, "preflight failed: "+err.Error(), o, err)
		}
		return step(j, StatePreflightPassed, "identity/eligibility revalidated on device", o)

	case StatePreflightPassed:
		return step(j, StateBackupRunning, "capturing mews evidence bundle", o)

	case StateBackupRunning:
		sha, err := acc.Backup(ctx)
		if err != nil {
			return fail(j, StateFailed, "backup failed: "+err.Error(), o, err)
		}
		return step(j, StateBackupCaptured, "backup bundle sha256="+short(sha), o)

	case StateBackupCaptured:
		// Persisting WriteInProgress here records write intent BEFORE any write.
		return step(j, StateWriteInProgress, "writing inactive slot", o)

	case StateWriteInProgress:
		if err := acc.WriteInactive(ctx); err != nil {
			// A failed inactive-slot write leaves the active slot bootable.
			return fail(j, StateFailed, "inactive-slot write failed: "+err.Error(), o, err)
		}
		if err := acc.VerifyWrite(ctx); err != nil {
			return fail(j, StateFailed, "write verification failed: "+err.Error(), o, err)
		}
		return step(j, StateWriteVerified, "inactive slot written and verified", o)

	case StateWriteVerified:
		return step(j, StateRebootPending, "pointing boot at the new slot and rebooting", o)

	case StateRebootPending:
		if err := acc.Reboot(ctx); err != nil {
			// Boot pointer/reboot ambiguity — hand off to manual recovery.
			return fail(j, StatePausedManual, "reboot failed: "+err.Error(), o, ErrManualRecovery)
		}
		return step(j, StateValidationRunning, "running post-change validation", o)

	case StateValidationRunning:
		if err := acc.Validate(ctx); err != nil {
			return fail(j, StatePausedManual, "post-change validation failed: "+err.Error(), o, ErrManualRecovery)
		}
		if o.Canary {
			if err := step(j, StateCanaryObservation, "holding for canary observation", o); err != nil {
				return err
			}
			return step(j, StateCompleted, "canary observation complete", o)
		}
		return step(j, StateCompleted, "operation complete", o)

	case StateCanaryObservation:
		return step(j, StateCompleted, "canary observation complete", o)

	default:
		return fmt.Errorf("executor: no action for state %s", j.State)
	}
}

// step advances the journal and persists it. A persist failure aborts the run —
// we must never proceed to a mutation whose intent we failed to record.
func step(j *Journal, to OpState, note string, o Options) error {
	if err := j.Advance(to, o.now(), note); err != nil {
		return err
	}
	if err := o.persist(j); err != nil {
		return fmt.Errorf("persist journal at %s: %w", to, err)
	}
	return nil
}

// fail records a terminal outcome (best effort) and returns cause.
func fail(j *Journal, to OpState, note string, o Options, cause error) error {
	if CanTransition(j.State, to) {
		if err := j.Advance(to, o.now(), note); err == nil {
			_ = o.persist(j)
		}
	}
	return cause
}
