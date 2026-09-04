// Package dump plans an on-device, verified full-flash capture — a safety net you
// take BEFORE any destructive flash. Where `mews` captures the small recovery-
// critical evidence bundle (mtd7/8/11 + config), `dump` captures the ENTIRE flash
// layout to device-local storage, hashes every artifact, and records a manifest,
// so a complete known-good image exists locally even if the network drops mid-job.
//
// Like the rest of swallow this package does no I/O: it parses `/proc/mtd`, plans
// the ordered read-only capture commands for an accessor to run, and verifies the
// result by hash ("prove, don't assume"). Two invariants it must never break:
//  1. It never emits a command that WRITES an mtd (`of=/dev/mtd*`); every command
//     is read-only. ART (RF calibration + factory MACs) is the true brick risk.
//  2. It refuses to place the dump under /dev or onto a raw flash device.
//
// ─────────────────────────────────────────────────────────────────────────────
// NOTES FOR THE NEXT AGENT (hardware pass) — please verify on a real ap-hk07:
//   - ap-hk07 is IPQ807x NAND. `dd if=/dev/mtdN` on NAND reads MAIN data only
//     (no OOB/ECC) and can choke on bad blocks. For a faithful NAND image prefer
//     `nanddump` — pass PlanOptions{NAND:true} to switch dd→nanddump. Confirm
//     which the device's busybox/tooling actually has (`which nanddump`).
//   - mtd numbering is NOT stable across firmware families. Always dump from a
//     freshly-parsed /proc/mtd, never hard-coded indices. The mtd7/8/11 roles
//     above are what we've seen on EWS/cloud images; re-verify per unit.
//   - UBI volumes (rootfs) may be presented as /dev/mtdN AND as ubi volumes.
//     Dumping the raw mtd is the safe superset; decoding UBI is out of scope here.
//   - destDir MUST be off the NAND being dumped (tmpfs /tmp, /dev/shm, or a USB
//     mount). Writing the dump onto the same NAND wastes space and risks the very
//     partitions you're capturing. SafeDestHint() flags a likely-unsafe dest; it
//     is a heuristic — the accessor should confirm the real mount + free space
//     from the `df` gate before proceeding.
//   - Space: EstimatedBytes() is the sum of partition sizes (compressed dumps are
//     smaller, but plan for the worst case). The `df` gate exists for this.
//   - Verify() currently compares on-device hashes to the pulled-to-host copy.
//     Also worth adding on hardware: a re-read of each mtd and hash-compare to
//     catch a partition that changed between capture and pull.
//
// ─────────────────────────────────────────────────────────────────────────────
package dump

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Partition is one entry parsed from /proc/mtd.
type Partition struct {
	Index     int    // N in mtdN
	Dev       string // /dev/mtdN
	Size      int64  // bytes
	EraseSize int64  // bytes
	Name      string // vendor label, e.g. "0:APPSBLENV"
}

// Role classifies a partition for criticality and human labeling.
type Role int

const (
	RoleOther  Role = iota
	RoleEnv         // APPSBLENV — the u-boot environment
	RoleBoot        // APPSBL / SBL / u-boot
	RoleART         // RF calibration + factory MACs — NEVER write; brick risk
	RoleKernel      // kernel image
	RoleRootfs      // rootfs / ubi
)

func (r Role) String() string {
	switch r {
	case RoleEnv:
		return "env"
	case RoleBoot:
		return "bootloader"
	case RoleART:
		return "art"
	case RoleKernel:
		return "kernel"
	case RoleRootfs:
		return "rootfs"
	default:
		return "other"
	}
}

// Classify maps a partition name to a role by substring (case-insensitive).
func Classify(name string) Role {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "art"):
		return RoleART
	case strings.Contains(n, "appsblenv"), strings.Contains(n, "env"):
		return RoleEnv
	case strings.Contains(n, "appsbl"), strings.Contains(n, "sbl"), strings.Contains(n, "u-boot"), strings.Contains(n, "uboot"):
		return RoleBoot
	case strings.Contains(n, "kernel"):
		return RoleKernel
	case strings.Contains(n, "rootfs"), strings.Contains(n, "ubi"):
		return RoleRootfs
	default:
		return RoleOther
	}
}

// Critical reports whether the partition is recovery-critical — its absence from
// a captured set must abort the pipeline. The env, bootloader, and ART partitions
// are what you cannot re-derive if lost.
func (p Partition) Critical() bool {
	switch Classify(p.Name) {
	case RoleART, RoleEnv, RoleBoot:
		return true
	default:
		return false
	}
}

// mtdLine matches `mtd7: 00080000 00020000 "0:APPSBLENV"`.
var mtdLine = regexp.MustCompile(`^mtd(\d+):\s+([0-9a-fA-F]+)\s+([0-9a-fA-F]+)\s+"(.*)"`)

// ParseProcMTD parses the contents of /proc/mtd into partitions, in device order.
// The header line ("dev: size erasesize name") and blank lines are ignored.
func ParseProcMTD(s string) ([]Partition, error) {
	var parts []Partition
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		m := mtdLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		size, err := strconv.ParseInt(m[2], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("mtd%d: bad size %q", idx, m[2])
		}
		erase, err := strconv.ParseInt(m[3], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("mtd%d: bad erasesize %q", idx, m[3])
		}
		parts = append(parts, Partition{
			Index:     idx,
			Dev:       fmt.Sprintf("/dev/mtd%d", idx),
			Size:      size,
			EraseSize: erase,
			Name:      m[4],
		})
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no mtd partitions found in /proc/mtd output")
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Index < parts[j].Index })
	return parts, nil
}

// Artifact is the on-device filename for a partition dump, e.g.
// "mtd7-0-appsblenv.bin". Deterministic and filesystem-safe.
func Artifact(p Partition) string {
	return fmt.Sprintf("mtd%d-%s.bin", p.Index, sanitize(p.Name))
}

var unsafeChars = regexp.MustCompile(`[^a-z0-9]+`)

func sanitize(name string) string {
	s := strings.ToLower(name)
	s = unsafeChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
