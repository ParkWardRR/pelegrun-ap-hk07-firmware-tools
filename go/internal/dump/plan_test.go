package dump

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func mustParts(t *testing.T) []Partition {
	t.Helper()
	parts, err := ParseProcMTD(procMTD)
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

func TestPlanIsReadOnly(t *testing.T) {
	steps, err := Plan(mustParts(t), "/tmp/swallow-dump", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// INVARIANT: no step may ever write an mtd.
	for _, s := range steps {
		if strings.Contains(s.Command, "of=/dev/mtd") || strings.Contains(s.Command, "> /dev/mtd") {
			t.Fatalf("dump plan must never write an mtd: %q", s.Command)
		}
	}
	// First two steps are gates (layout + space), and every partition is dumped.
	if !steps[0].Gate || !steps[1].Gate {
		t.Fatal("first two steps must be gates (proc-mtd, df)")
	}
	dumped := 0
	for _, s := range steps {
		if strings.HasPrefix(s.Command, "dd if=/dev/mtd") {
			dumped++
		}
	}
	if dumped != 6 {
		t.Fatalf("expected 6 partition dumps, got %d", dumped)
	}
	// Last step hashes everything (a verify gate).
	last := steps[len(steps)-1]
	if !strings.Contains(last.Command, "sha256sum") || !last.Gate {
		t.Fatalf("final step must hash artifacts as a gate: %+v", last)
	}
}

func TestPlanNANDUsesNanddump(t *testing.T) {
	steps, err := Plan(mustParts(t), "/tmp/d", PlanOptions{NAND: true})
	if err != nil {
		t.Fatal(err)
	}
	sawNand := false
	for _, s := range steps {
		if strings.HasPrefix(s.Command, "nanddump") {
			sawNand = true
		}
		if strings.HasPrefix(s.Command, "dd if=/dev/mtd") {
			t.Fatal("NAND plan should use nanddump, not dd")
		}
	}
	if !sawNand {
		t.Fatal("NAND plan should emit nanddump commands")
	}
}

func TestPlanDestRefusals(t *testing.T) {
	cases := []struct {
		dest string
		want error
	}{
		{"", ErrNoDest},
		{"relative/path", ErrDestNotAbs},
		{"/dev/mtd11", ErrDestIsDevice},
		{"/dev", ErrDestIsDevice},
	}
	for _, tc := range cases {
		if _, err := Plan(mustParts(t), tc.dest, PlanOptions{}); !errors.Is(err, tc.want) {
			t.Errorf("Plan(dest=%q) err=%v, want %v", tc.dest, err, tc.want)
		}
	}
}

func TestSafeDestHint(t *testing.T) {
	if ok, _ := SafeDestHint("/tmp/swallow-dump"); !ok {
		t.Fatal("/tmp should be hinted safe")
	}
	if ok, _ := SafeDestHint("/dev/shm/x"); !ok {
		t.Fatal("/dev/shm should be hinted safe")
	}
	if ok, note := SafeDestHint("/overlay/backups"); ok || note == "" {
		t.Fatal("an on-flash path should be flagged unsafe with a reason")
	}
}

func TestEstimatedBytesAndMissingCritical(t *testing.T) {
	parts := mustParts(t)
	// Sum of the hex sizes above.
	if got := EstimatedBytes(parts); got <= 0 {
		t.Fatalf("estimated bytes should be positive, got %d", got)
	}
	// Capture everything except ART (mtd11) -> ART reported missing.
	captured := map[string]bool{}
	for _, p := range parts {
		if p.Index != 11 {
			captured[Artifact(p)] = true
		}
	}
	miss := MissingCritical(parts, captured)
	if len(miss) != 1 || !strings.Contains(miss[0], "art") {
		t.Fatalf("expected ART missing, got %v", miss)
	}
}

func TestManifestRoundTripAndVerify(t *testing.T) {
	parts := mustParts(t)
	m := NewManifest(parts, "/tmp/swallow-dump", "swallow 0.4.0", PlanOptions{Model: "ap-hk07", Serial: "SWLWX420001Q"})
	if m.Validation != StatusPlanned || len(m.Partitions) != 6 {
		t.Fatalf("bad initial manifest: %+v", m)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// Build matching on-device + host hash sets -> verify passes.
	onDev := map[string]string{}
	host := map[string]string{}
	for _, p := range parts {
		onDev[Artifact(p)] = "abc123"
		host[Artifact(p)] = "ABC123" // case-insensitive match
	}
	if err := got.Verify(onDev, host); err != nil {
		t.Fatalf("verify should pass on matching hashes: %v", err)
	}
	if got.Validation != StatusVerified {
		t.Fatal("verify should advance validation to verified")
	}

	// Corrupt one host hash -> verify fails.
	host[Artifact(parts[0])] = "deadbeef"
	m2 := NewManifest(parts, "/tmp/d", "t", PlanOptions{})
	if err := m2.Verify(onDev, host); err == nil {
		t.Fatal("verify should fail on a hash mismatch")
	}
}
