package fleet

import (
	"fmt"
	"sort"
)

// Cohort names a device's rollout wave in a plan.
const (
	CohortCanary  = "canary"
	CohortMain    = "main"
	CohortBlocked = "" // not eligible; not scheduled
)

// SelectCanary chooses the canary cohort from eligible devices. Selection is
// deterministic (sorted by fleet ID) so the same inputs always yield the same
// cohort — a canary must be reproducible, not a random first-N.
//
// Size is CanaryPolicy.Count when > 0, else ceil(Percent% of eligible) with a
// floor of one when Percent > 0. With DiverseByModel, the cohort is filled round
// robin across models (by the models map) so it represents fleet diversity rather
// than clustering on whichever model sorts first.
func SelectCanary(eligible []string, models map[string]string, pol CanaryPolicy) []string {
	ids := append([]string(nil), eligible...)
	sort.Strings(ids)
	n := canarySize(len(ids), pol)
	if n <= 0 {
		return nil
	}
	if n >= len(ids) {
		return ids
	}
	if !pol.DiverseByModel {
		return append([]string(nil), ids[:n]...)
	}
	return diversePick(ids, models, n)
}

func canarySize(total int, pol CanaryPolicy) int {
	if total == 0 {
		return 0
	}
	if pol.Count > 0 {
		if pol.Count > total {
			return total
		}
		return pol.Count
	}
	if pol.Percent <= 0 {
		return 0
	}
	// ceil(total*percent/100), at least 1.
	n := (total*pol.Percent + 99) / 100
	if n < 1 {
		n = 1
	}
	if n > total {
		n = total
	}
	return n
}

// diversePick walks per-model queues (each already fleet-ID sorted) round robin,
// taking one at a time until n are chosen. Model order is itself sorted for
// determinism. Devices with no model entry fall under "" and participate too.
func diversePick(ids []string, models map[string]string, n int) []string {
	byModel := map[string][]string{}
	for _, id := range ids {
		m := models[id]
		byModel[m] = append(byModel[m], id)
	}
	modelKeys := make([]string, 0, len(byModel))
	for m := range byModel {
		modelKeys = append(modelKeys, m)
	}
	sort.Strings(modelKeys)

	picked := make([]string, 0, n)
	for len(picked) < n {
		progressed := false
		for _, m := range modelKeys {
			q := byModel[m]
			if len(q) == 0 {
				continue
			}
			picked = append(picked, q[0])
			byModel[m] = q[1:]
			progressed = true
			if len(picked) == n {
				break
			}
		}
		if !progressed {
			break
		}
	}
	sort.Strings(picked)
	return picked
}

// GateOutcome is the canary health-gate verdict.
type GateOutcome string

const (
	GatePromote GateOutcome = "promote" // proceed to the wider fleet
	GateHold    GateOutcome = "hold"    // observation incomplete; wait
	GateAbort   GateOutcome = "abort"   // stop new work; investigate/recover
)

// CohortResult is the observed outcome of a cohort.
type CohortResult struct {
	Total   int
	Passed  int
	Failed  int
	Unknown int
}

func (r CohortResult) terminal() int { return r.Passed + r.Failed + r.Unknown }

// EvaluateGate decides whether a cohort may promote. The error budget is checked
// first: exceeding it aborts regardless of pass rate. Until every device reaches a
// terminal state the gate holds. Once terminal, it promotes only if the pass rate
// meets the policy bar; otherwise it aborts (failures are within budget but the
// cohort did not clear the bar, so widening the blast radius is not justified).
func EvaluateGate(r CohortResult, pol CanaryPolicy, budget ErrorBudget) (GateOutcome, string) {
	if r.Failed > budget.MaxFailed {
		return GateAbort, fmt.Sprintf("failed %d exceeds error budget %d", r.Failed, budget.MaxFailed)
	}
	if r.Unknown > budget.MaxUnknown {
		return GateAbort, fmt.Sprintf("unknown %d exceeds error budget %d", r.Unknown, budget.MaxUnknown)
	}
	if r.Total == 0 {
		return GateHold, "empty cohort"
	}
	if r.terminal() < r.Total {
		return GateHold, fmt.Sprintf("observation incomplete: %d/%d terminal", r.terminal(), r.Total)
	}
	passRate := float64(r.Passed) / float64(r.Total)
	if passRate < pol.MinPassRate {
		return GateAbort, fmt.Sprintf("pass rate %.2f below required %.2f", passRate, pol.MinPassRate)
	}
	return GatePromote, fmt.Sprintf("pass rate %.2f meets required %.2f", passRate, pol.MinPassRate)
}
