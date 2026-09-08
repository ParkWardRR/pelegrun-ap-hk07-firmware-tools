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
	want := []string{"ap-hk07", "ap-hk07-openwrt"} // Flashable() sorts
	if len(flashable) != len(want) {
		t.Fatalf("flashable = %v, want %v", flashable, want)
	}
	for i, m := range want {
		if flashable[i] != m {
			t.Fatalf("flashable = %v, want %v", flashable, want)
		}
	}
}

func TestDefaultRegistryHasOpenWrtTarget(t *testing.T) {
	r := DefaultRegistry()
	ow := r.Get("ap-hk07-openwrt")
	if ow == nil {
		t.Fatal("ap-hk07-openwrt must be in the default registry")
	}
	if ow.Has(CapFlashAB) {
		t.Fatal("the OpenWrt target must not declare flash_ab — it has no A/B fallback")
	}
	if !ow.Has(CapFlashFixedPartition) {
		t.Fatal("the OpenWrt target must declare flash_fixed_partition")
	}
	if err := ow.CanFlash(); err != nil {
		t.Fatalf("ap-hk07-openwrt should be flash-ready: %v", err)
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
