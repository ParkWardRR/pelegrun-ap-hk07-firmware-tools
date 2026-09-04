package adapter

import (
	"errors"
	"testing"
)

func fullCaps() map[Capability]bool {
	m := map[Capability]bool{}
	for _, c := range AllCapabilities {
		m[c] = true
	}
	return m
}

func TestVerifiedFullAdapterValidates(t *testing.T) {
	s := Support{Adapter: "ap-hk07", Model: "ap-hk07", Tier: TierVerified, Capabilities: fullCaps()}
	if errs := s.Validate(); len(errs) != 0 {
		t.Fatalf("a fully-capable verified adapter should validate, got %v", errs)
	}
	if err := s.CanFlash(); err != nil {
		t.Fatalf("verified full adapter should be flash-ready, got %v", err)
	}
}

func TestCanFlashRequiresBackupAndRecovery(t *testing.T) {
	caps := fullCaps()
	delete(caps, CapBackup)
	s := Support{Tier: TierExperimental, Capabilities: caps}
	if err := s.CanFlash(); err == nil {
		t.Fatal("no backup capability must forbid flashing")
	}

	caps = fullCaps()
	delete(caps, CapUARTRecovery)
	delete(caps, CapTFTPRecovery)
	s = Support{Tier: TierVerified, Capabilities: caps}
	if err := s.CanFlash(); err == nil {
		t.Fatal("no recovery route must forbid flashing")
	}
}

func TestCandidateAndRecoveryOnlyCannotFlash(t *testing.T) {
	if err := (Support{Tier: TierCandidate, Capabilities: fullCaps()}).CanFlash(); err == nil {
		t.Fatal("candidate tier must not flash even with full capabilities")
	}
	if err := (Support{Tier: TierRecoveryOnly, Capabilities: fullCaps()}).CanFlash(); err == nil {
		t.Fatal("recovery-only tier must not flash")
	}
}

func TestValidateFlagsTierWithoutFlashReadiness(t *testing.T) {
	// Declares verified but cannot actually flash -> inconsistent.
	caps := fullCaps()
	delete(caps, CapFlashAB)
	errs := Support{Tier: TierVerified, Capabilities: caps}.Validate()
	if len(errs) == 0 {
		t.Fatal("verified-but-not-flash-ready must be flagged")
	}
}

func TestValidateUnknownCapability(t *testing.T) {
	s := Support{Tier: TierCandidate, Capabilities: map[Capability]bool{"teleport": true}}
	errs := s.Validate()
	found := false
	for _, e := range errs {
		var un ErrUnknownCapability
		if errors.As(e, &un) {
			found = true
		}
	}
	if !found {
		t.Fatalf("unknown capability key should be flagged, got %v", errs)
	}
}

func TestRecoveryOnlyNeedsARecoveryRoute(t *testing.T) {
	s := Support{Tier: TierRecoveryOnly, Capabilities: map[Capability]bool{CapInspect: true}}
	if len(s.Validate()) == 0 {
		t.Fatal("recovery-only with no recovery capability should be flagged")
	}
	ok := Support{Tier: TierRecoveryOnly, Capabilities: map[Capability]bool{CapInspect: true, CapTFTPRecovery: true}}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("recovery-only with tftp should validate, got %v", errs)
	}
}

func TestMissingAndList(t *testing.T) {
	s := Support{Tier: TierCandidate, Capabilities: map[Capability]bool{CapInspect: true, CapFingerprint: true}}
	if got := len(s.CapabilityList()); got != 2 {
		t.Fatalf("capability list = %d, want 2", got)
	}
	if got := len(s.Missing()); got != len(AllCapabilities)-2 {
		t.Fatalf("missing = %d, want %d", got, len(AllCapabilities)-2)
	}
}
