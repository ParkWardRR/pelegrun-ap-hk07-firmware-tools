package fleet

import (
	"encoding/json"
	"os"
	"sort"
)

// DeviceOutcome summarizes one device's operation for a rollout report.
type DeviceOutcome struct {
	FleetID string  `json:"fleet_id"`
	State   OpState `json:"state"`
	// Classified rollup of State for health-gate math.
	Class string `json:"class"` // "passed" | "failed" | "unknown" | "in_flight"
}

// RolloutReport aggregates per-device journals into a structured operator summary
// plus the canary health-gate verdict. It is the P9e observability artifact: a
// fleet failure needs machine-readable evidence, not just scrolled-back logs.
type RolloutReport struct {
	Total     int             `json:"total"`
	Completed int             `json:"completed"`
	Failed    int             `json:"failed"`
	Manual    int             `json:"manual_recovery"`
	InFlight  int             `json:"in_flight"`
	Outcomes  []DeviceOutcome `json:"outcomes"`
	// Gate is the canary verdict computed from these outcomes; empty Reason when
	// no gate was evaluated (e.g. a non-canary batch report).
	Gate       GateOutcome `json:"gate,omitempty"`
	GateReason string      `json:"gate_reason,omitempty"`
}

// classify maps a journal state to a health-gate class. A completed op passed; a
// failed or manual-recovery op counts against the error budget as failed/unknown;
// anything still moving is in-flight (observation incomplete).
func classify(s OpState) string {
	switch s {
	case StateCompleted:
		return "passed"
	case StateFailed:
		return "failed"
	case StatePausedManual:
		return "unknown" // needs a human; not a clean failure, not a pass
	default:
		return "in_flight"
	}
}

// BuildReport aggregates journals into a report. Outcomes are sorted by fleet ID
// for stable output. It does not evaluate the gate — call EvaluateReportGate when
// the batch is a canary cohort.
func BuildReport(journals []*Journal) *RolloutReport {
	r := &RolloutReport{Total: len(journals)}
	for _, j := range journals {
		class := classify(j.State)
		switch class {
		case "passed":
			r.Completed++
		case "failed":
			r.Failed++
		case "unknown":
			r.Manual++
		default:
			r.InFlight++
		}
		r.Outcomes = append(r.Outcomes, DeviceOutcome{FleetID: j.FleetID, State: j.State, Class: class})
	}
	sort.Slice(r.Outcomes, func(i, j int) bool { return r.Outcomes[i].FleetID < r.Outcomes[j].FleetID })
	return r
}

// CohortResult projects the report onto the canary gate's counting model:
// passed = completed, failed = failed, unknown = manual-recovery (terminal but
// unclear). In-flight devices are deliberately NOT counted in any terminal
// bucket, so the gate sees terminal() < Total and holds until observation is
// complete — an operation still moving can never read as a pass.
func (r *RolloutReport) CohortResult() CohortResult {
	return CohortResult{
		Total:   r.Total,
		Passed:  r.Completed,
		Failed:  r.Failed,
		Unknown: r.Manual,
	}
}

// EvaluateReportGate runs the canary health gate over the report and records the
// verdict on it, returning the same outcome for convenience.
func (r *RolloutReport) EvaluateReportGate(pol CanaryPolicy, budget ErrorBudget) GateOutcome {
	out, reason := EvaluateGate(r.CohortResult(), pol, budget)
	r.Gate = out
	r.GateReason = reason
	return out
}

// Save writes the report JSON.
func (r *RolloutReport) Save(path string) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
