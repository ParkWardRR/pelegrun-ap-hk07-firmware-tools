package mews

import "testing"

func TestPlanCoversCriticalMTDs(t *testing.T) {
	want := map[string]bool{"mtd7-appsblenv.bin": false, "mtd8-appsbl.bin": false, "mtd11-art.bin": false, "env-good.txt": false}
	for _, a := range Plan() {
		if _, ok := want[a.Name]; ok {
			want[a.Name] = true
			if !a.Critical {
				t.Errorf("%s must be critical", a.Name)
			}
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("plan is missing %s", name)
		}
	}
}

func TestArtReadOnly(t *testing.T) {
	for _, a := range Plan() {
		if a.Name == "mtd11-art.bin" && !contains(a.Command, "if=/dev/mtd11") {
			t.Fatalf("ART capture must be a read (dd if=): %q", a.Command)
		}
		if contains(a.Command, "of=/dev/mtd") {
			t.Fatalf("backup plan must never WRITE an mtd: %q", a.Command)
		}
	}
}

func TestBundleName(t *testing.T) {
	got := BundleName("2026-09-02", "Scuderia Toro Rosso", "88:DC:97:04:44:07")
	if got != "2026-09-02_Scuderia-Toro-Rosso_88-dc-97-04-44-07" {
		t.Fatalf("bundle = %q", got)
	}
}

func TestMissingCritical(t *testing.T) {
	captured := map[string]bool{"proc-mtd.txt": true, "env-good.txt": true, "mtd7-appsblenv.bin": true, "mtd8-appsbl.bin": true}
	miss := MissingCritical(captured) // mtd11-art missing
	if len(miss) != 1 || miss[0] != "mtd11-art.bin" {
		t.Fatalf("missing = %v", miss)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
