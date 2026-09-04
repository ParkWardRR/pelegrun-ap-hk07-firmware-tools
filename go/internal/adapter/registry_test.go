package adapter

import (
	"path/filepath"
	"testing"
)

func TestDefaultRegistryIsConsistent(t *testing.T) {
	r := DefaultRegistry()
	if probs := r.Validate(); len(probs) != 0 {
		t.Fatalf("builtin registry must be internally consistent, got %v", probs)
	}
	aphk := r.Get("AP-HK07") // case-insensitive
	if aphk == nil {
		t.Fatal("ap-hk07 must be in the default registry")
	}
	if aphk.Tier != TierExperimental {
		t.Fatalf("ap-hk07 should be experimental (not yet hardware-verified), got %s", aphk.Tier)
	}
	if aphk.CanFlash() != nil {
		t.Fatalf("ap-hk07 should be flash-capable at experimental: %v", aphk.CanFlash())
	}
}

func TestFlashableList(t *testing.T) {
	r := DefaultRegistry()
	// Add a candidate board that must NOT be flashable.
	r.Adapters = append(r.Adapters, Support{
		Adapter: "other", Model: "senao-xyz", Tier: TierCandidate,
		Capabilities: map[Capability]bool{CapFingerprint: true},
	})
	flashable := r.Flashable()
	if len(flashable) != 1 || flashable[0] != "ap-hk07" {
		t.Fatalf("only ap-hk07 should be flashable, got %v", flashable)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	r, err := LoadRegistry(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Get("ap-hk07") == nil {
		t.Fatal("missing file should fall back to the builtin default")
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reg.json")
	if err := DefaultRegistry().Save(path); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Validate()) != 0 {
		t.Fatal("round-tripped registry should still validate")
	}
}
