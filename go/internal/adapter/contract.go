// Package adapter defines the P12a capability contract every board adapter must
// declare before it can be trusted with a destructive path. Capabilities are
// explicit and typed: an adapter must never imply support for an operation it
// cannot verify or recover. The contract is pure data + rules — no device I/O —
// so a new board's declared support can be checked and gated in CI and at
// plan time (ROADMAP P12, "adapter contract and qualification").
package adapter

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is one operation an adapter may support.
type Capability string

const (
	CapFingerprint  Capability = "fingerprint"   // structured model/board/identity evidence
	CapInspect      Capability = "inspect"       // read-only state collection
	CapAccess       Capability = "access"        // ssh/cloud/luci route + error classification
	CapBackup       Capability = "backup"        // capture regions/config with hashes
	CapProvision    Capability = "provision"     // serial/append-only behavior
	CapFlashAB      Capability = "flash_ab"      // inactive-slot write + verify + rollback
	CapUARTRecovery Capability = "uart_recovery" // gated serial console repair
	CapTFTPRecovery Capability = "tftp_recovery" // bootloader/TFTP recovery
	CapHealthCheck  Capability = "health_check"  // model-appropriate post-op validation
	CapEvidence     Capability = "evidence"      // structured before/after diagnostic bundle
)

// AllCapabilities is the closed set, in a stable order for reporting.
var AllCapabilities = []Capability{
	CapFingerprint, CapInspect, CapAccess, CapBackup, CapProvision,
	CapFlashAB, CapUARTRecovery, CapTFTPRecovery, CapHealthCheck, CapEvidence,
}

// Tier is a board's support level. Destructive paths are gated on it.
type Tier string

const (
	TierCandidate    Tier = "candidate"     // identified only; no destructive recommendation
	TierExperimental Tier = "experimental"  // demonstrated on limited hardware; caveats disclosed
	TierVerified     Tier = "verified"      // passed the qualification matrix
	TierRecoveryOnly Tier = "recovery-only" // inspect/recover only; no normal flashing
	TierUnsupported  Tier = "unsupported"   // refuse destructive operations by default
)

func (t Tier) rank() int {
	switch t {
	case TierVerified:
		return 4
	case TierExperimental:
		return 3
	case TierRecoveryOnly:
		return 2
	case TierCandidate:
		return 1
	case TierUnsupported:
		return 0
	default:
		return 0
	}
}

// Support is a board adapter's self-declared capability record. It is the unit a
// model registry (P12d) stores and CI validates.
type Support struct {
	Adapter       string              `json:"adapter"`
	Model         string              `json:"model"`
	BoardRevision string              `json:"board_revision,omitempty"`
	Tier          Tier                `json:"tier"`
	Capabilities  map[Capability]bool `json:"capabilities"`
	Evidence      []string            `json:"evidence,omitempty"`
}

// Has reports whether the adapter declares capability c.
func (s Support) Has(c Capability) bool { return s.Capabilities[c] }

// Missing returns the capabilities the adapter does NOT declare, in stable order.
func (s Support) Missing() []Capability {
	var out []Capability
	for _, c := range AllCapabilities {
		if !s.Capabilities[c] {
			out = append(out, c)
		}
	}
	return out
}

// ErrUnknownCapability is returned by Validate for a capability key outside the
// closed set — a typo must not silently read as "unsupported".
type ErrUnknownCapability struct{ Cap Capability }

func (e ErrUnknownCapability) Error() string {
	return fmt.Sprintf("unknown capability %q (not in the contract)", e.Cap)
}

// Validate checks the record's internal consistency: every declared capability
// key is known, the tier is a real tier, and the declared capabilities are
// coherent with the tier (a destructive tier must actually declare the
// capabilities that make a destructive path recoverable). It returns every
// problem found.
func (s Support) Validate() []error {
	var errs []error

	for c := range s.Capabilities {
		if !known(c) {
			errs = append(errs, ErrUnknownCapability{Cap: c})
		}
	}
	if s.Tier.rank() == 0 && s.Tier != TierUnsupported {
		errs = append(errs, fmt.Errorf("unknown tier %q", s.Tier))
	}

	// A tier that permits flashing must be able to prove and recover it.
	if s.Tier == TierExperimental || s.Tier == TierVerified {
		if err := s.CanFlash(); err != nil {
			errs = append(errs, fmt.Errorf("tier %q but not flash-ready: %w", s.Tier, err))
		}
	}
	// Recovery-only must at least be able to inspect and recover.
	if s.Tier == TierRecoveryOnly && !(s.Has(CapUARTRecovery) || s.Has(CapTFTPRecovery)) {
		errs = append(errs, fmt.Errorf("recovery-only tier declares no recovery capability"))
	}
	return errs
}

// CanFlash reports whether the adapter is permitted to run a destructive A/B
// flash: it must be at least experimental, and must declare backup, inactive-slot
// flash, and at least one recovery route — an adapter can never offer a
// destructive path it cannot back up or recover from.
func (s Support) CanFlash() error {
	if s.Tier.rank() < TierExperimental.rank() || s.Tier == TierRecoveryOnly {
		return fmt.Errorf("tier %q does not permit flashing", s.Tier)
	}
	var missing []string
	if !s.Has(CapBackup) {
		missing = append(missing, string(CapBackup))
	}
	if !s.Has(CapFlashAB) {
		missing = append(missing, string(CapFlashAB))
	}
	if !s.Has(CapUARTRecovery) && !s.Has(CapTFTPRecovery) {
		missing = append(missing, "a recovery route (uart_recovery or tftp_recovery)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required capabilities: %s", strings.Join(missing, ", "))
	}
	return nil
}

func known(c Capability) bool {
	for _, k := range AllCapabilities {
		if k == c {
			return true
		}
	}
	return false
}

// CapabilityList returns the declared capabilities in stable order (for display).
func (s Support) CapabilityList() []Capability {
	var out []Capability
	for _, c := range AllCapabilities {
		if s.Capabilities[c] {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
