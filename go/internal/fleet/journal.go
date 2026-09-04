package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// OpState is a device operation's position in its durable lifecycle. This state
// machine is distinct from terminal output: it is persisted before every
// irreversible action so an interrupted run can be reconciled, never blindly
// replayed. States mirror the ROADMAP "durable operation journal" table.
type OpState string

const (
	StatePlanned           OpState = "planned"
	StatePreflightPassed   OpState = "preflight_passed"
	StateBackupRunning     OpState = "backup_running"
	StateBackupCaptured    OpState = "backup_captured"
	StateWriteInProgress   OpState = "inactive_slot_write_in_progress"
	StateWriteVerified     OpState = "write_verified"
	StateRebootPending     OpState = "reboot_pending"
	StateValidationRunning OpState = "post_change_validation_running"
	StateCanaryObservation OpState = "canary_observation"
	StateCompleted         OpState = "completed"
	StatePausedManual      OpState = "paused_manual_recovery"
	StateFailed            OpState = "failed"
)

// transitions is the legal forward adjacency. Every non-terminal state may also
// diverge to failed or paused_manual_recovery (added by allowedNext); those two
// and completed are terminal for automation and require a new plan to proceed.
var transitions = map[OpState][]OpState{
	StatePlanned:           {StatePreflightPassed},
	StatePreflightPassed:   {StateBackupRunning},
	StateBackupRunning:     {StateBackupCaptured},
	StateBackupCaptured:    {StateWriteInProgress},
	StateWriteInProgress:   {StateWriteVerified},
	StateWriteVerified:     {StateRebootPending},
	StateRebootPending:     {StateValidationRunning},
	StateValidationRunning: {StateCanaryObservation, StateCompleted},
	StateCanaryObservation: {StateCompleted},
	StateCompleted:         {},
	StatePausedManual:      {},
	StateFailed:            {},
}

// IsTerminal reports whether a state permits no further automated transition.
func IsTerminal(s OpState) bool {
	switch s {
	case StateCompleted, StateFailed, StatePausedManual:
		return true
	default:
		return false
	}
}

// AllowedNext returns the states reachable from s (forward progress plus the
// divergences to failed/paused available from any live state). The result is a
// fresh slice safe for the caller to mutate.
func AllowedNext(s OpState) []OpState {
	if IsTerminal(s) {
		return nil
	}
	fwd := transitions[s]
	out := make([]OpState, 0, len(fwd)+2)
	out = append(out, fwd...)
	out = append(out, StateFailed, StatePausedManual)
	return out
}

// CanTransition reports whether from -> to is legal.
func CanTransition(from, to OpState) bool {
	for _, n := range AllowedNext(from) {
		if n == to {
			return true
		}
	}
	return false
}

// Transition is one recorded step in a journal.
type Transition struct {
	From OpState `json:"from"`
	To   OpState `json:"to"`
	At   string  `json:"at"` // RFC3339
	Note string  `json:"note,omitempty"`
}

// Journal is a device operation's durable record. It pins the operation to the
// exact policy/plan/target/identity it was authorized against, so a resumed run
// can prove it is still acting on the same decision.
type Journal struct {
	SchemaVersion int          `json:"schema_version"`
	OperationID   string       `json:"operation_id"`
	FleetID       string       `json:"fleet_id"`
	ToolVersion   string       `json:"tool_version"`
	Actor         string       `json:"actor,omitempty"`
	Adapter       string       `json:"adapter,omitempty"`
	PolicyHash    string       `json:"policy_hash,omitempty"`
	PlanHash      string       `json:"plan_hash,omitempty"`
	TargetHash    string       `json:"target_hash,omitempty"`
	IdentityHash  string       `json:"identity_hash,omitempty"`
	State         OpState      `json:"state"`
	Transitions   []Transition `json:"transitions"`
}

// NewJournal starts a journal in the Planned state with an opening transition.
func NewJournal(operationID, fleetID, toolVersion string, now time.Time) *Journal {
	at := now.UTC().Format(time.RFC3339)
	return &Journal{
		SchemaVersion: SchemaVersion,
		OperationID:   operationID,
		FleetID:       fleetID,
		ToolVersion:   toolVersion,
		State:         StatePlanned,
		Transitions:   []Transition{{From: "", To: StatePlanned, At: at}},
	}
}

// ErrIllegalTransition is returned by Advance for a disallowed state change.
type ErrIllegalTransition struct {
	From, To OpState
}

func (e ErrIllegalTransition) Error() string {
	return fmt.Sprintf("illegal operation transition %s -> %s", e.From, e.To)
}

// Advance moves the journal to state to, recording the transition. It fails
// closed on an illegal move; the caller must persist the journal (Save) before
// performing the irreversible action the new state represents.
func (j *Journal) Advance(to OpState, now time.Time, note string) error {
	if !CanTransition(j.State, to) {
		return ErrIllegalTransition{From: j.State, To: to}
	}
	j.Transitions = append(j.Transitions, Transition{
		From: j.State,
		To:   to,
		At:   now.UTC().Format(time.RFC3339),
		Note: note,
	})
	j.State = to
	return nil
}

// Ref summarizes the journal for the inventory's latest_operation pointer.
func (j *Journal) Ref() OperationRef {
	return OperationRef{
		OperationID:   j.OperationID,
		State:         j.State,
		JournalSHA256: mustHash(j),
	}
}

// Save writes the journal JSON.
func (j *Journal) Save(path string) error {
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadJournal reads a journal JSON.
func LoadJournal(path string) (*Journal, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var j Journal
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	return &j, nil
}
