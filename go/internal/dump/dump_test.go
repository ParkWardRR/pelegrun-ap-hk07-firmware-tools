package dump

import (
	"testing"
)

// A representative IPQ807x /proc/mtd. Exact sizes/names must be re-verified on
// real hardware (see package NOTES); this exercises the parser and planner.
const procMTD = `dev:    size   erasesize  name
mtd0: 00100000 00020000 "0:SBL1"
mtd7: 00080000 00020000 "0:APPSBLENV"
mtd8: 00140000 00020000 "0:APPSBL"
mtd9: 00080000 00020000 "0:KERNEL"
mtd10: 04000000 00020000 "rootfs"
mtd11: 00080000 00020000 "0:ART"
`

func TestParseProcMTD(t *testing.T) {
	parts, err := ParseProcMTD(procMTD)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 6 {
		t.Fatalf("expected 6 partitions, got %d", len(parts))
	}
	// Parsed in index order; hex sizes decoded.
	if parts[0].Index != 0 || parts[0].Name != "0:SBL1" {
		t.Fatalf("first partition wrong: %+v", parts[0])
	}
	var art Partition
	for _, p := range parts {
		if p.Index == 11 {
			art = p
		}
	}
	if art.Dev != "/dev/mtd11" || art.Size != 0x80000 || art.EraseSize != 0x20000 {
		t.Fatalf("ART partition parsed wrong: %+v", art)
	}
	if Classify(art.Name) != RoleART || !art.Critical() {
		t.Fatal("ART must classify as art and be critical")
	}
}

func TestParseProcMTDEmpty(t *testing.T) {
	if _, err := ParseProcMTD("dev:    size   erasesize  name\n"); err == nil {
		t.Fatal("a header-only /proc/mtd should error (no partitions)")
	}
}

func TestClassifyAndCritical(t *testing.T) {
	cases := map[string]Role{
		"0:APPSBLENV": RoleEnv,
		"0:APPSBL":    RoleBoot,
		"0:ART":       RoleART,
		"0:KERNEL":    RoleKernel,
		"rootfs":      RoleRootfs,
		"ubi_rootfs":  RoleRootfs,
		"0:SBL1":      RoleBoot, // contains "sbl"
		"0:MIBIB":     RoleOther,
	}
	for name, want := range cases {
		if got := Classify(name); got != want {
			t.Errorf("Classify(%q)=%v want %v", name, got, want)
		}
	}
	// env/boot/art are critical; kernel/rootfs/other are not.
	if !(Partition{Name: "0:ART"}).Critical() || (Partition{Name: "rootfs"}).Critical() {
		t.Fatal("criticality wrong")
	}
}

func TestArtifactNaming(t *testing.T) {
	p := Partition{Index: 7, Name: "0:APPSBLENV"}
	if got := Artifact(p); got != "mtd7-0-appsblenv.bin" {
		t.Fatalf("artifact name = %q", got)
	}
}
