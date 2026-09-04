package fleet

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Window is an optional maintenance window; apply may only start inside it.
type Window struct {
	StartRFC3339 string `json:"start"`
	EndRFC3339   string `json:"end"`
}

// Open reports whether now is within the window (an unset bound is open-ended;
// an unparseable bound is treated as closed, failing safe).
func (w *Window) Open(now time.Time) bool {
	if w == nil {
		return true
	}
	if w.StartRFC3339 != "" {
		start, err := time.Parse(time.RFC3339, w.StartRFC3339)
		if err != nil || now.Before(start) {
			return false
		}
	}
	if w.EndRFC3339 != "" {
		end, err := time.Parse(time.RFC3339, w.EndRFC3339)
		if err != nil || now.After(end) {
			return false
		}
	}
	return true
}

// ErrorBudget stops new rollout work once too many devices end in a bad state.
type ErrorBudget struct {
	MaxFailed  int `json:"max_failed"`
	MaxUnknown int `json:"max_unknown"`
}

// CanaryPolicy configures staged rollout: cohort size, observation, and the pass
// bar. Count wins when > 0; otherwise Percent of the eligible set (at least one).
type CanaryPolicy struct {
	Count                 int     `json:"count,omitempty"`
	Percent               int     `json:"percent,omitempty"`
	MinObservationSeconds int     `json:"min_observation_seconds,omitempty"`
	MinPassRate           float64 `json:"min_pass_rate,omitempty"`
	DiverseByModel        bool    `json:"diverse_by_model,omitempty"`
}

// Policy is the fleet rule set. It is hashed into every plan; a changed policy
// invalidates a previously built plan at apply time.
type Policy struct {
	Version               string             `json:"version"`
	SupportedModels       []string           `json:"supported_models,omitempty"` // empty = any
	AllowedFamilies       []string           `json:"allowed_families,omitempty"` // eyas family strings; empty = any
	RequireBackup         bool               `json:"require_backup"`
	MinIdentityConfidence IdentityConfidence `json:"min_identity_confidence,omitempty"` // default verified
	StaleAfterSeconds     int                `json:"stale_after_seconds,omitempty"`
	MaxConcurrency        int                `json:"max_concurrency,omitempty"`
	PerNetworkConcurrency int                `json:"per_network_concurrency,omitempty"`
	MaintenanceWindow     *Window            `json:"maintenance_window,omitempty"`
	Canary                CanaryPolicy       `json:"canary"`
	ErrorBudget           ErrorBudget        `json:"error_budget"`
}

// Hash is the canonical digest a plan pins the policy by.
func (p Policy) Hash() string { return mustHash(p) }

func (p Policy) minConfidence() IdentityConfidence {
	if p.MinIdentityConfidence == "" {
		return ConfidenceVerified
	}
	return p.MinIdentityConfidence
}

// Decision is a per-device eligibility outcome. Reasons list the blocks when
// Eligible is false (and are empty when eligible).
type Decision struct {
	Eligible bool     `json:"eligible"`
	Reasons  []string `json:"reasons,omitempty"`
}

// Evaluate decides whether a single device may be part of a destructive
// operation for tgt, as of now. It is deterministic and never mutates anything.
// Every failing rule is reported so an operator can see the full picture, not
// just the first block.
func (p Policy) Evaluate(d Device, tgt Target, now time.Time) Decision {
	var reasons []string

	// A conflicting identity is an absolute stop, regardless of other policy.
	if d.Physical.Confidence == ConfidenceConflict {
		reasons = append(reasons, "identity in conflict: reading contradicts the record")
	} else if d.Physical.Confidence.rank() < p.minConfidence().rank() {
		reasons = append(reasons, fmt.Sprintf("identity confidence %q below required %q",
			nonEmptyConf(d.Physical.Confidence), p.minConfidence()))
	}

	if len(p.SupportedModels) > 0 && !containsFold(p.SupportedModels, d.Physical.Model) {
		reasons = append(reasons, fmt.Sprintf("model %q not in supported set %v", d.Physical.Model, p.SupportedModels))
	}

	if len(p.AllowedFamilies) > 0 && d.Software.FirmwareFamily != "" && !containsFold(p.AllowedFamilies, d.Software.FirmwareFamily) {
		reasons = append(reasons, fmt.Sprintf("firmware family %q not in allowed set %v", d.Software.FirmwareFamily, p.AllowedFamilies))
	}

	if p.StaleAfterSeconds > 0 && d.IsStale(now, time.Duration(p.StaleAfterSeconds)*time.Second) {
		reasons = append(reasons, fmt.Sprintf("discovery reading is stale (older than %ds)", p.StaleAfterSeconds))
	}

	if p.RequireBackup && d.Evidence.BackupBundleSHA256 == "" {
		reasons = append(reasons, "no backup evidence bundle recorded (mews) — required before mutation")
	}

	sort.Strings(reasons)
	return Decision{Eligible: len(reasons) == 0, Reasons: reasons}
}

func nonEmptyConf(c IdentityConfidence) IdentityConfidence {
	if c == "" {
		return ConfidenceUnknown
	}
	return c
}

func containsFold(set []string, v string) bool {
	for _, s := range set {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
