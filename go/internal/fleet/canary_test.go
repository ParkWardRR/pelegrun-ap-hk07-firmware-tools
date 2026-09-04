package fleet

import (
	"reflect"
	"testing"
)

func TestSelectCanaryCountAndDeterminism(t *testing.T) {
	elig := []string{"d", "a", "c", "b"}
	got := SelectCanary(elig, nil, CanaryPolicy{Count: 2})
	want := []string{"a", "b"} // sorted, first two
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("count cohort = %v, want %v", got, want)
	}
	// Deterministic regardless of input order.
	if !reflect.DeepEqual(SelectCanary([]string{"b", "d", "a", "c"}, nil, CanaryPolicy{Count: 2}), want) {
		t.Fatal("cohort selection must be order-independent")
	}
}

func TestSelectCanaryPercentCeil(t *testing.T) {
	elig := []string{"a", "b", "c", "d", "e"}
	// 10% of 5 = 0.5 -> ceil 1
	if got := SelectCanary(elig, nil, CanaryPolicy{Percent: 10}); len(got) != 1 {
		t.Fatalf("10%% of 5 should be 1, got %v", got)
	}
	// 50% of 5 = 2.5 -> ceil 3
	if got := SelectCanary(elig, nil, CanaryPolicy{Percent: 50}); len(got) != 3 {
		t.Fatalf("50%% of 5 should be 3, got %v", got)
	}
}

func TestSelectCanaryDiverseByModel(t *testing.T) {
	elig := []string{"a1", "a2", "a3", "b1"}
	models := map[string]string{"a1": "modelA", "a2": "modelA", "a3": "modelA", "b1": "modelB"}
	got := SelectCanary(elig, models, CanaryPolicy{Count: 2, DiverseByModel: true})
	// Round robin across sorted models: modelA->a1, modelB->b1.
	want := []string{"a1", "b1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diverse cohort = %v, want %v (one per model before doubling up)", got, want)
	}
}

func TestSelectCanaryEmpty(t *testing.T) {
	if got := SelectCanary(nil, nil, CanaryPolicy{Count: 3}); got != nil {
		t.Fatalf("no eligible devices -> nil cohort, got %v", got)
	}
	if got := SelectCanary([]string{"a"}, nil, CanaryPolicy{}); got != nil {
		t.Fatalf("no count/percent -> nil cohort, got %v", got)
	}
}

func TestEvaluateGate(t *testing.T) {
	pol := CanaryPolicy{MinPassRate: 0.8}
	budget := ErrorBudget{MaxFailed: 1, MaxUnknown: 0}

	if o, _ := EvaluateGate(CohortResult{Total: 3, Passed: 1}, pol, budget); o != GateHold {
		t.Fatalf("incomplete observation should hold, got %s", o)
	}
	if o, _ := EvaluateGate(CohortResult{Total: 5, Passed: 5}, pol, budget); o != GatePromote {
		t.Fatalf("all passed should promote, got %s", o)
	}
	if o, _ := EvaluateGate(CohortResult{Total: 5, Passed: 3, Failed: 2}, pol, budget); o != GateAbort {
		t.Fatalf("failures over budget should abort, got %s", o)
	}
	if o, _ := EvaluateGate(CohortResult{Total: 5, Passed: 3, Unknown: 1, Failed: 1}, pol, budget); o != GateAbort {
		t.Fatalf("unknowns over budget should abort, got %s", o)
	}
	// Terminal, within budget, but below pass bar -> abort (don't widen blast radius).
	if o, _ := EvaluateGate(CohortResult{Total: 5, Passed: 3, Failed: 2}, CanaryPolicy{MinPassRate: 0.8}, ErrorBudget{MaxFailed: 2, MaxUnknown: 0}); o != GateAbort {
		t.Fatalf("0.6 pass under 0.8 bar (failures within budget) should abort, got %s", o)
	}
	// Exactly at the bar with failures within budget promotes.
	if o, _ := EvaluateGate(CohortResult{Total: 10, Passed: 8, Failed: 1, Unknown: 1}, CanaryPolicy{MinPassRate: 0.8}, ErrorBudget{MaxFailed: 2, MaxUnknown: 2}); o != GatePromote {
		t.Fatalf("0.8 pass within budget should promote, got %s", o)
	}
}
