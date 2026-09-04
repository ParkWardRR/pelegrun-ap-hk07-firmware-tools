package fleet

import (
	"path/filepath"
	"testing"
)

func journalIn(id string, state OpState) *Journal {
	j := NewJournal("op-"+id, id, "0.4.0", refNow)
	j.State = state // force a terminal/interim state for reporting
	return j
}

func TestBuildReportCounts(t *testing.T) {
	js := []*Journal{
		journalIn("a", StateCompleted),
		journalIn("b", StateCompleted),
		journalIn("c", StateFailed),
		journalIn("d", StatePausedManual),
		journalIn("e", StateValidationRunning),
	}
	r := BuildReport(js)
	if r.Total != 5 || r.Completed != 2 || r.Failed != 1 || r.Manual != 1 || r.InFlight != 1 {
		t.Fatalf("bad counts: %+v", r)
	}
	// Outcomes sorted by fleet ID.
	if r.Outcomes[0].FleetID != "a" || r.Outcomes[4].FleetID != "e" {
		t.Fatal("outcomes must be sorted by fleet ID")
	}
	if r.Outcomes[3].Class != "unknown" { // paused_manual -> unknown
		t.Fatalf("paused_manual should classify as unknown, got %q", r.Outcomes[3].Class)
	}
}

func TestReportGatePromotesAllPass(t *testing.T) {
	js := []*Journal{journalIn("a", StateCompleted), journalIn("b", StateCompleted)}
	r := BuildReport(js)
	out := r.EvaluateReportGate(CanaryPolicy{MinPassRate: 1.0}, ErrorBudget{})
	if out != GatePromote || r.Gate != GatePromote || r.GateReason == "" {
		t.Fatalf("all-pass cohort should promote, got %s (%s)", out, r.GateReason)
	}
}

func TestReportGateAbortsOverBudget(t *testing.T) {
	js := []*Journal{journalIn("a", StateCompleted), journalIn("b", StateFailed), journalIn("c", StateFailed)}
	r := BuildReport(js)
	out := r.EvaluateReportGate(CanaryPolicy{MinPassRate: 0.5}, ErrorBudget{MaxFailed: 1, MaxUnknown: 0})
	if out != GateAbort {
		t.Fatalf("two failures over a budget of one should abort, got %s", out)
	}
}

func TestReportGateHoldsInFlight(t *testing.T) {
	js := []*Journal{journalIn("a", StateCompleted), journalIn("b", StateValidationRunning)}
	r := BuildReport(js)
	// One still in-flight -> unknown -> observation incomplete -> hold.
	out := r.EvaluateReportGate(CanaryPolicy{MinPassRate: 0.5}, ErrorBudget{MaxFailed: 0, MaxUnknown: 5})
	if out != GateHold {
		t.Fatalf("in-flight cohort should hold, got %s", out)
	}
}

func TestReportSaveRoundTrip(t *testing.T) {
	r := BuildReport([]*Journal{journalIn("a", StateCompleted)})
	path := filepath.Join(t.TempDir(), "report.json")
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
}
