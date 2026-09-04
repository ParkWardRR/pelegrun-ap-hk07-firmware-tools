package fleet

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Observed is a fresh discovery reading for one device (from eyas/jess/mews),
// keyed to a fleet record by FleetID. Reconcile folds it into the record.
type Observed struct {
	FleetID           string
	Model             string
	BoardRevision     string
	HardwareSerial    string
	MACAddresses      []string
	FirmwareFamily    string
	FirmwareVersion   string
	FirmwareSHA256    string
	ActiveSlot        string
	InactiveSlot      string
	BootloaderVersion string
	FitVersion        string
	ManagementAddress string
	Adapter           string
	SnapshotSHA256    string
	SeenAt            time.Time
}

// Reconciliation is the result of folding an Observed reading into a record.
type Reconciliation struct {
	Device          Device   // the updated record
	Changed         []string // human-readable software/access changes applied
	Mismatch        bool     // a STABLE identity field contradicts the record
	MismatchReasons []string // why the physical identity is in conflict
}

// Reconcile updates prev with a fresh reading. Software identity and access are
// expected to change and are folded in freely; physical identity is stable, so a
// changed model, hardware serial, or a fully disjoint MAC set is treated as a
// different unit answering at the same address — reported as a mismatch and the
// record's confidence is dropped to Conflict. The caller decides whether to
// accept a mismatched record; a plan must never mutate a Conflict device.
func Reconcile(prev Device, obs Observed) Reconciliation {
	r := Reconciliation{Device: prev}
	d := &r.Device
	if d.FleetID == "" {
		d.FleetID = obs.FleetID
	}

	// Stable identity: detect contradictions before trusting the reading.
	if prev.Physical.Model != "" && obs.Model != "" && !strings.EqualFold(prev.Physical.Model, obs.Model) {
		r.Mismatch = true
		r.MismatchReasons = append(r.MismatchReasons, fmt.Sprintf("model %q -> %q", prev.Physical.Model, obs.Model))
	}
	if prev.Physical.HardwareSerial != "" && obs.HardwareSerial != "" && !strings.EqualFold(prev.Physical.HardwareSerial, obs.HardwareSerial) {
		r.Mismatch = true
		r.MismatchReasons = append(r.MismatchReasons, fmt.Sprintf("hardware_serial %q -> %q", prev.Physical.HardwareSerial, obs.HardwareSerial))
	}
	if len(prev.Physical.MACAddresses) > 0 && len(obs.MACAddresses) > 0 && disjointMACs(prev.Physical.MACAddresses, obs.MACAddresses) {
		r.Mismatch = true
		r.MismatchReasons = append(r.MismatchReasons, fmt.Sprintf("mac set %v shares nothing with %v", prev.Physical.MACAddresses, obs.MACAddresses))
	}

	// Fill/refresh physical identity fields from the reading.
	if obs.Model != "" {
		d.Physical.Model = obs.Model
	}
	if obs.BoardRevision != "" {
		d.Physical.BoardRevision = obs.BoardRevision
	}
	if obs.HardwareSerial != "" {
		d.Physical.HardwareSerial = obs.HardwareSerial
	}
	if len(obs.MACAddresses) > 0 {
		d.Physical.MACAddresses = normMACs(obs.MACAddresses)
	}

	// Confidence: a contradiction wins; otherwise a matched hardware serial is
	// verification, and anything else is at most probable.
	switch {
	case r.Mismatch:
		d.Physical.Confidence = ConfidenceConflict
	case prev.Physical.HardwareSerial != "" && obs.HardwareSerial != "" && strings.EqualFold(prev.Physical.HardwareSerial, obs.HardwareSerial):
		d.Physical.Confidence = ConfidenceVerified
	case obs.HardwareSerial != "":
		if d.Physical.Confidence.rank() < ConfidenceProbable.rank() {
			d.Physical.Confidence = ConfidenceProbable
		}
	default:
		if d.Physical.Confidence == "" {
			d.Physical.Confidence = ConfidenceUnknown
		}
	}

	// Software identity: fold every populated field, recording real changes.
	setSoft(&d.Software.FirmwareFamily, obs.FirmwareFamily, "firmware_family", &r.Changed)
	setSoft(&d.Software.FirmwareVersion, obs.FirmwareVersion, "firmware_version", &r.Changed)
	setSoft(&d.Software.FirmwareSHA256, obs.FirmwareSHA256, "firmware_sha256", &r.Changed)
	setSoft(&d.Software.ActiveSlot, obs.ActiveSlot, "active_slot", &r.Changed)
	setSoft(&d.Software.InactiveSlot, obs.InactiveSlot, "inactive_slot", &r.Changed)
	setSoft(&d.Software.BootloaderVersion, obs.BootloaderVersion, "bootloader_version", &r.Changed)
	setSoft(&d.Software.FitVersion, obs.FitVersion, "fit_version", &r.Changed)

	// Access is always refreshed.
	if obs.ManagementAddress != "" {
		d.Access.ManagementAddress = obs.ManagementAddress
	}
	if obs.Adapter != "" {
		d.Access.Adapter = obs.Adapter
	}
	if obs.SnapshotSHA256 != "" {
		d.Access.DiscoverySnapshotSHA256 = obs.SnapshotSHA256
	}
	if !obs.SeenAt.IsZero() {
		d.Access.LastSeenAt = obs.SeenAt.UTC().Format(time.RFC3339)
	}
	return r
}

func setSoft(dst *string, val, field string, changed *[]string) {
	if val == "" || *dst == val {
		return
	}
	if *dst != "" {
		*changed = append(*changed, fmt.Sprintf("%s %q -> %q", field, *dst, val))
	}
	*dst = val
}

func normMACs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, m := range in {
		out = append(out, strings.ToLower(strings.TrimSpace(m)))
	}
	sort.Strings(out)
	return out
}

func disjointMACs(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, m := range a {
		set[strings.ToLower(strings.TrimSpace(m))] = true
	}
	for _, m := range b {
		if set[strings.ToLower(strings.TrimSpace(m))] {
			return false
		}
	}
	return true
}
