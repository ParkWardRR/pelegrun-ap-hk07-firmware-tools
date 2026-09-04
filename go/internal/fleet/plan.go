package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"
)

// Target is the firmware/adoption target a plan is built against. It is hashed
// into the plan; a changed target invalidates the plan at apply time.
type Target struct {
	ImageName      string `json:"image_name"`
	ImageSHA256    string `json:"image_sha256"`
	FirmwareFamily string `json:"firmware_family,omitempty"`
	FitVersion     string `json:"fit_version,omitempty"`
}

// Hash is the canonical digest a plan pins the target by.
func (t Target) Hash() string { return mustHash(t) }

// PlanEntry is one device's plan-time decision, including the identity pin that
// apply revalidates and the slot the flash would write (always the inactive one).
type PlanEntry struct {
	FleetID      string   `json:"fleet_id"`
	Model        string   `json:"model,omitempty"`
	Address      string   `json:"address,omitempty"`
	IdentityHash string   `json:"identity_hash"`
	Eligible     bool     `json:"eligible"`
	Reasons      []string `json:"reasons,omitempty"`
	Cohort       string   `json:"cohort,omitempty"`
	ActiveSlot   string   `json:"active_slot,omitempty"`
	InactiveSlot string   `json:"inactive_slot,omitempty"`
}

// Plan is an immutable, read-only rollout plan. Building one changes no device.
// PlanHash covers every field except itself, so any post-hoc edit is detectable.
type Plan struct {
	SchemaVersion int         `json:"schema_version"`
	CreatedAt     string      `json:"created_at"`
	ToolVersion   string      `json:"tool_version"`
	PolicyVersion string      `json:"policy_version"`
	PolicyHash    string      `json:"policy_hash"`
	Target        Target      `json:"target"`
	TargetHash    string      `json:"target_hash"`
	InventoryHash string      `json:"inventory_hash"`
	WindowOpen    bool        `json:"window_open"`
	Entries       []PlanEntry `json:"entries"`
	EligibleCount int         `json:"eligible_count"`
	BlockedCount  int         `json:"blocked_count"`
	CanaryCount   int         `json:"canary_count"`
	PlanHash      string      `json:"plan_hash"`
}

// BuildPlan produces the deterministic read-only plan. It performs discovery-free
// evaluation only — all device state must already be in the inventory (run a
// reconcile pass first). Entries are sorted by fleet ID; eligible devices are then
// split into canary and main cohorts per policy.
func BuildPlan(f *Fleet, p Policy, tgt Target, toolVersion string, now time.Time) *Plan {
	devs := append([]Device(nil), f.Devices...)
	sort.Slice(devs, func(i, j int) bool { return devs[i].FleetID < devs[j].FleetID })

	pl := &Plan{
		SchemaVersion: SchemaVersion,
		CreatedAt:     now.UTC().Format(time.RFC3339),
		ToolVersion:   toolVersion,
		PolicyVersion: p.Version,
		PolicyHash:    p.Hash(),
		Target:        tgt,
		TargetHash:    tgt.Hash(),
		InventoryHash: f.Snapshot(),
		WindowOpen:    p.MaintenanceWindow.Open(now),
	}

	var eligibleIDs []string
	models := map[string]string{}
	for _, d := range devs {
		dec := p.Evaluate(d, tgt, now)
		e := PlanEntry{
			FleetID:      d.FleetID,
			Model:        d.Physical.Model,
			Address:      d.Access.ManagementAddress,
			IdentityHash: physicalIdentityHash(d.Physical),
			Eligible:     dec.Eligible,
			Reasons:      dec.Reasons,
			ActiveSlot:   d.Software.ActiveSlot,
			InactiveSlot: d.Software.InactiveSlot,
		}
		if dec.Eligible {
			pl.EligibleCount++
			eligibleIDs = append(eligibleIDs, d.FleetID)
			models[d.FleetID] = d.Physical.Model
		} else {
			pl.BlockedCount++
		}
		pl.Entries = append(pl.Entries, e)
	}

	canarySet := map[string]bool{}
	for _, id := range SelectCanary(eligibleIDs, models, p.Canary) {
		canarySet[id] = true
	}
	pl.CanaryCount = len(canarySet)
	for i := range pl.Entries {
		if !pl.Entries[i].Eligible {
			continue
		}
		if canarySet[pl.Entries[i].FleetID] {
			pl.Entries[i].Cohort = CohortCanary
		} else {
			pl.Entries[i].Cohort = CohortMain
		}
	}

	pl.PlanHash = ""
	pl.PlanHash = mustHash(pl)
	return pl
}

// Save writes the plan JSON.
func (pl *Plan) Save(path string) error {
	b, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadPlan reads a plan JSON.
func LoadPlan(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pl Plan
	if err := json.Unmarshal(b, &pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

// recomputeHash returns what PlanHash should be for the current field values.
func (pl *Plan) recomputeHash() string {
	saved := pl.PlanHash
	pl.PlanHash = ""
	h := mustHash(pl)
	pl.PlanHash = saved
	return h
}

// Apply-guard failures. Each is a refusal, never a warning: apply is the one
// place that turns a plan into destructive action, so it must fail closed.
var (
	ErrPlanTampered      = errors.New("plan hash does not match its contents (plan was modified)")
	ErrPolicyChanged     = errors.New("policy changed since the plan was built")
	ErrTargetChanged     = errors.New("target image changed since the plan was built")
	ErrWindowClosed      = errors.New("outside the maintenance window")
	ErrDeviceMissing     = errors.New("an eligible planned device is no longer in the inventory")
	ErrIdentityMismatch  = errors.New("an eligible planned device's physical identity changed")
	ErrDiscoveryStale    = errors.New("an eligible planned device's discovery reading is stale")
	ErrEligibilityChange = errors.New("a planned device is no longer eligible under current policy")
)

// ApplyGuard revalidates a plan against the live inventory, current policy, and
// target immediately before execution. It returns the first refusal it finds, or
// nil when the plan is safe to run. It deliberately does NOT require whole-fleet
// hash equality — last_seen and other benign fields change constantly — it instead
// revalidates each eligible device: still present, identity unchanged (the pin),
// not stale, and still eligible. It must be called with the same p/tgt the plan
// was built for; a mismatch is reported rather than silently accepted.
func ApplyGuard(pl *Plan, f *Fleet, p Policy, tgt Target, now time.Time, maxAge time.Duration) error {
	if pl.PlanHash == "" || pl.PlanHash != pl.recomputeHash() {
		return ErrPlanTampered
	}
	if pl.PolicyHash != p.Hash() {
		return fmt.Errorf("%w (plan %s, current %s)", ErrPolicyChanged, short(pl.PolicyHash), short(p.Hash()))
	}
	if pl.TargetHash != tgt.Hash() {
		return fmt.Errorf("%w (plan %s, current %s)", ErrTargetChanged, short(pl.TargetHash), short(tgt.Hash()))
	}
	if !p.MaintenanceWindow.Open(now) {
		return ErrWindowClosed
	}
	for _, e := range pl.Entries {
		if !e.Eligible {
			continue
		}
		d := f.Get(e.FleetID)
		if d == nil {
			return fmt.Errorf("%w: %s", ErrDeviceMissing, e.FleetID)
		}
		if physicalIdentityHash(d.Physical) != e.IdentityHash {
			return fmt.Errorf("%w: %s", ErrIdentityMismatch, e.FleetID)
		}
		if d.Physical.Confidence == ConfidenceConflict {
			return fmt.Errorf("%w: %s (identity in conflict)", ErrIdentityMismatch, e.FleetID)
		}
		if maxAge > 0 && d.IsStale(now, maxAge) {
			return fmt.Errorf("%w: %s", ErrDiscoveryStale, e.FleetID)
		}
		if dec := p.Evaluate(*d, tgt, now); !dec.Eligible {
			return fmt.Errorf("%w: %s (%v)", ErrEligibilityChange, e.FleetID, dec.Reasons)
		}
	}
	return nil
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
