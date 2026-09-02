// Package mews builds the per-device backup/evidence bundle — the shelter you can
// always return the bird to. It plans the (read-only) capture commands and names
// the bundle deterministically; running the commands is the accessor's job.
//
// Constitution III: the bundle is a HARD gate before any destructive step, and
// ART (mtd11) is read-only-backup, never write.
package mews

import (
	"fmt"
	"strings"
)

// Artifact is one file the backup must contain, with how to capture it.
type Artifact struct {
	Name     string // file name inside the bundle
	Command  string // read-only command that produces it (over the accessor)
	Critical bool   // if true, the pipeline must abort when this is missing
}

// Plan is the ordered set of read-only captures for a device.
// mtd7 = APPSBLENV (env), mtd8 = APPSBL (bootloader+defaults), mtd11 = ART
// (RF calibration + factory MACs — the true brick risk if ever written).
func Plan() []Artifact {
	return []Artifact{
		{Name: "proc-mtd.txt", Command: "cat /proc/mtd", Critical: true},
		{Name: "env-good.txt", Command: "fw_printenv", Critical: true},
		{Name: "mtd7-appsblenv.bin", Command: "dd if=/dev/mtd7 bs=64k", Critical: true},
		{Name: "mtd8-appsbl.bin", Command: "dd if=/dev/mtd8 bs=64k", Critical: true},
		{Name: "mtd11-art.bin", Command: "dd if=/dev/mtd11 bs=64k", Critical: true},
		{Name: "dmesg.txt", Command: "dmesg", Critical: false},
	}
}

// BundleName is the deterministic, sortable bundle directory for a device.
// Example: 2026-09-02_ScuderiaToroRosso_88-DC-97-04-44-07
func BundleName(date, asset, mac string) string {
	safeMac := strings.NewReplacer(":", "-", " ", "").Replace(strings.ToLower(mac))
	safeAsset := strings.NewReplacer(" ", "-", "/", "-").Replace(asset)
	return fmt.Sprintf("%s_%s_%s", date, safeAsset, safeMac)
}

// Manifest records what was captured and its integrity hashes.
type Manifest struct {
	Bundle    string            `json:"bundle"`
	Device    string            `json:"device"`
	MAC       string            `json:"mac"`
	Capdate   string            `json:"captured"`
	SHA256    map[string]string `json:"sha256"` // artifact name -> hex digest
	Firmware  string            `json:"firmware,omitempty"`
	SrcImage  string            `json:"src_image_sha256,omitempty"`
	PatchedTo string            `json:"patched_product_id,omitempty"`
}

// MissingCritical returns the critical artifacts absent from a captured set —
// the pipeline must refuse to continue if this is non-empty.
func MissingCritical(captured map[string]bool) []string {
	var miss []string
	for _, a := range Plan() {
		if a.Critical && !captured[a.Name] {
			miss = append(miss, a.Name)
		}
	}
	return miss
}
