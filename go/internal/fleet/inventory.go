// Package fleet is the P9 control plane: it composes swallow's safe single-device
// primitives (eyas/mews/hood/band/flash) into a policy- and evidence-driven batch
// workflow. Like the rest of swallow it is pure logic that does no device I/O — it
// builds a normalized inventory, a deterministic read-only plan, canary cohorts,
// and a durable per-device operation journal for an accessor to execute.
//
// The core safety stance (ROADMAP "Scope guarantee"): a fleet workflow must never
// become an unbounded destructive loop. Planning changes nothing; apply refuses a
// modified plan, a changed policy/target, a stale reading, or an identity mismatch.
package fleet

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
)

// SchemaVersion is the on-disk version for the inventory, plan, and journal
// documents. Bump it (with a migration or an explicit refusal) on any breaking
// field change — release engineering track in the ROADMAP.
const SchemaVersion = 1

// IdentityConfidence grades how sure we are that a record names the physical
// unit currently answering at its address.
type IdentityConfidence string

const (
	ConfidenceVerified IdentityConfidence = "verified" // matched a stable hardware identifier
	ConfidenceProbable IdentityConfidence = "probable" // consistent, not yet hardware-confirmed
	ConfidenceUnknown  IdentityConfidence = "unknown"  // no identity evidence yet
	ConfidenceConflict IdentityConfidence = "conflict" // observed identity contradicts the record
)

func (c IdentityConfidence) rank() int {
	switch c {
	case ConfidenceVerified:
		return 3
	case ConfidenceProbable:
		return 2
	case ConfidenceUnknown:
		return 1
	case ConfidenceConflict:
		return 0
	default:
		return 1
	}
}

// EligibilityState is a device's cached plan-time eligibility.
type EligibilityState string

const (
	EligibilityUnknown  EligibilityState = "unknown"
	EligibilityEligible EligibilityState = "eligible"
	EligibilityBlocked  EligibilityState = "blocked"
)

// PhysicalIdentity is the stable, hardware-bound identity of a unit. Changes here
// mean a different physical device, not an update — the reconciler flags them.
type PhysicalIdentity struct {
	Model          string             `json:"model"`
	BoardRevision  string             `json:"board_revision,omitempty"`
	MACAddresses   []string           `json:"mac_addresses,omitempty"`
	HardwareSerial string             `json:"hardware_serial,omitempty"`
	Confidence     IdentityConfidence `json:"identity_confidence"`
}

// SoftwareIdentity is the mutable firmware/slot state; it changes every flash.
type SoftwareIdentity struct {
	FirmwareFamily    string `json:"firmware_family,omitempty"`
	FirmwareVersion   string `json:"firmware_version,omitempty"`
	FirmwareSHA256    string `json:"firmware_sha256,omitempty"`
	ActiveSlot        string `json:"active_slot,omitempty"`
	InactiveSlot      string `json:"inactive_slot,omitempty"`
	BootloaderVersion string `json:"bootloader_version,omitempty"`
	FitVersion        string `json:"fit_version,omitempty"`
}

// Access is where and how to reach the device — none of it is stable identity.
type Access struct {
	ManagementAddress       string `json:"management_address,omitempty"`
	Adapter                 string `json:"adapter,omitempty"`
	LastSeenAt              string `json:"last_seen_at,omitempty"` // RFC3339
	DiscoverySnapshotSHA256 string `json:"discovery_snapshot_sha256,omitempty"`
}

// Evidence records safety artifacts a policy can require before mutation.
type Evidence struct {
	BackupBundleSHA256 string `json:"backup_bundle_sha256,omitempty"`
	BackupCapturedAt   string `json:"backup_captured_at,omitempty"` // RFC3339
}

// OperationRef points at the device's most recent operation journal.
type OperationRef struct {
	OperationID   string  `json:"operation_id,omitempty"`
	State         OpState `json:"state,omitempty"`
	JournalSHA256 string  `json:"journal_sha256,omitempty"`
}

// Eligibility caches the last plan-time decision for a device.
type Eligibility struct {
	State         EligibilityState `json:"state"`
	PolicyVersion string           `json:"policy_version,omitempty"`
	Reasons       []string         `json:"reasons,omitempty"`
}

// Device is one fleet member. FleetID is the operator-assigned stable key — never
// a mutable identifier like DHCP address, hostname, or UI label.
type Device struct {
	FleetID         string           `json:"fleet_id"`
	Physical        PhysicalIdentity `json:"physical_identity"`
	Software        SoftwareIdentity `json:"software_identity"`
	Access          Access           `json:"access"`
	Evidence        Evidence         `json:"evidence"`
	Eligibility     Eligibility      `json:"eligibility"`
	LatestOperation OperationRef     `json:"latest_operation,omitempty"`
}

// Fleet is the normalized inventory document.
type Fleet struct {
	SchemaVersion int      `json:"schema_version"`
	Devices       []Device `json:"devices"`
}

// Load reads a fleet inventory JSON (a missing file is an empty fleet).
func Load(path string) (*Fleet, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Fleet{SchemaVersion: SchemaVersion}, nil
	}
	if err != nil {
		return nil, err
	}
	var f Fleet
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.SchemaVersion == 0 {
		f.SchemaVersion = SchemaVersion
	}
	return &f, nil
}

// Save writes the fleet inventory JSON.
func (f *Fleet) Save(path string) error {
	if f.SchemaVersion == 0 {
		f.SchemaVersion = SchemaVersion
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Get returns a pointer to the device with the given fleet ID, or nil.
func (f *Fleet) Get(fleetID string) *Device {
	for i := range f.Devices {
		if f.Devices[i].FleetID == fleetID {
			return &f.Devices[i]
		}
	}
	return nil
}

// Snapshot returns the canonical SHA-256 of the whole inventory, for reporting
// and change-detection. (Apply does NOT require whole-fleet hash equality —
// last_seen legitimately changes constantly — it revalidates per device instead.)
func (f *Fleet) Snapshot() string { return mustHash(f) }

// physicalIdentityHash pins the stable identity fields only. A change here across
// a plan/apply boundary is an identity mismatch and must abort the apply.
func physicalIdentityHash(p PhysicalIdentity) string {
	macs := append([]string(nil), p.MACAddresses...)
	sort.Strings(macs)
	return mustHash(struct {
		Model  string   `json:"model"`
		Serial string   `json:"serial"`
		MACs   []string `json:"macs"`
	}{strings.ToLower(p.Model), strings.ToLower(p.HardwareSerial), lowerAll(macs)})
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

// LastSeen parses the device's last-seen timestamp.
func (d Device) LastSeen() (time.Time, bool) {
	if d.Access.LastSeenAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, d.Access.LastSeenAt)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// IsStale reports whether the last discovery reading is older than maxAge (or
// missing). A stale reading must not be trusted to gate a destructive change.
func (d Device) IsStale(now time.Time, maxAge time.Duration) bool {
	if maxAge <= 0 {
		return false
	}
	t, ok := d.LastSeen()
	if !ok {
		return true
	}
	return now.Sub(t) > maxAge
}
