package dump

import (
	"errors"
	"fmt"
	"strings"
)

// Step is one ordered, read-only capture command for the accessor to run.
type Step struct {
	Desc     string // human-readable purpose
	Command  string // the exact shell command (always read-only w.r.t. flash)
	Artifact string // file it produces under destDir (empty if none)
	Critical bool   // absence must abort the pipeline
	Gate     bool   // a hard gate: verify success before continuing
}

// PlanOptions tune the capture.
type PlanOptions struct {
	NAND   bool   // use nanddump instead of dd (NAND bad-block/ECC handling)
	Model  string // for the manifest
	Serial string // for the manifest
}

// Dump-plan refusals — each fails closed rather than risking a bad capture.
var (
	ErrNoDest       = errors.New("destination directory required")
	ErrDestNotAbs   = errors.New("destination must be an absolute path")
	ErrDestIsDevice = errors.New("refusing to place the dump under /dev (raw flash device)")
	ErrNoPartitions = errors.New("no partitions to dump")
)

// Plan builds the ordered on-device capture plan: record the layout, check free
// space at the destination, dump every partition read-only, then hash everything.
// It never emits a command that writes an mtd. destDir is a device-local path
// (tmpfs/USB) — see SafeDestHint and the package NOTES for placement guidance.
func Plan(parts []Partition, destDir string, opts PlanOptions) ([]Step, error) {
	if err := ValidateDest(destDir); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, ErrNoPartitions
	}
	dest := strings.TrimRight(destDir, "/")

	steps := []Step{
		{
			Desc:     "record the flash layout",
			Command:  fmt.Sprintf("cat /proc/mtd > %s/proc-mtd.txt", dest),
			Artifact: "proc-mtd.txt",
			Critical: true,
			Gate:     true,
		},
		{
			Desc: "check free space at the destination",
			// The accessor must confirm free space >= EstimatedBytes before dumping.
			Command:  fmt.Sprintf("df -k %s", dest),
			Artifact: "df.txt",
			Critical: true,
			Gate:     true,
		},
	}

	for _, p := range parts {
		steps = append(steps, Step{
			Desc:     fmt.Sprintf("dump mtd%d (%s, %s)", p.Index, p.Name, Classify(p.Name)),
			Command:  dumpCmd(p, dest, opts.NAND),
			Artifact: Artifact(p),
			Critical: p.Critical(),
		})
	}

	// Hash everything on-device so the manifest can be filled and later verified.
	steps = append(steps, Step{
		Desc:     "hash every artifact (integrity baseline)",
		Command:  fmt.Sprintf("cd %s && sha256sum *.bin proc-mtd.txt > SHA256SUMS", dest),
		Artifact: "SHA256SUMS",
		Critical: true,
		Gate:     true,
	})

	return steps, nil
}

// dumpCmd returns the read-only capture command for one partition. It reads from
// /dev/mtdN and writes to destDir — never the reverse.
func dumpCmd(p Partition, dest string, nand bool) string {
	out := fmt.Sprintf("%s/%s", dest, Artifact(p))
	if nand {
		// nanddump handles bad blocks / ECC; --omitoob keeps the image comparable
		// to a plain read. Verify the exact flags on the device's mtd-utils.
		return fmt.Sprintf("nanddump --omitoob -f %s %s", out, p.Dev)
	}
	bs := p.EraseSize
	if bs <= 0 {
		bs = 65536
	}
	return fmt.Sprintf("dd if=%s of=%s bs=%d", p.Dev, out, bs)
}

// ValidateDest enforces the placement safety rules.
func ValidateDest(dest string) error {
	if strings.TrimSpace(dest) == "" {
		return ErrNoDest
	}
	if !strings.HasPrefix(dest, "/") {
		return ErrDestNotAbs
	}
	if dest == "/dev" || strings.HasPrefix(dest, "/dev/") {
		return ErrDestIsDevice
	}
	return nil
}

// SafeDestHint reports whether dest looks like volatile/removable storage that is
// safe to hold a dump (tmpfs, ramdisk, USB/SD mount). It is a heuristic only — the
// accessor must still confirm the real mountpoint and free space from the `df`
// gate. Returns (false, reason) when the location looks like on-flash storage.
func SafeDestHint(dest string) (bool, string) {
	safePrefixes := []string{"/tmp", "/dev/shm", "/run", "/mnt", "/media", "/var/tmp"}
	for _, pre := range safePrefixes {
		if dest == pre || strings.HasPrefix(dest, pre+"/") {
			return true, fmt.Sprintf("%s looks like volatile/removable storage", pre)
		}
	}
	return false, "destination is not under a known volatile/removable mount (/tmp, /dev/shm, /run, /mnt, /media); confirm it is NOT on the NAND being dumped"
}

// EstimatedBytes is the total capture size (sum of partition sizes). Actual dumps
// may be smaller, but plan free space for this worst case.
func EstimatedBytes(parts []Partition) int64 {
	var total int64
	for _, p := range parts {
		total += p.Size
	}
	return total
}

// MissingCritical returns the recovery-critical partitions absent from a captured
// artifact set (artifact name -> captured). The pipeline must refuse to continue —
// and certainly must not flash — when this is non-empty.
func MissingCritical(parts []Partition, captured map[string]bool) []string {
	var miss []string
	for _, p := range parts {
		if p.Critical() && !captured[Artifact(p)] {
			miss = append(miss, Artifact(p))
		}
	}
	return miss
}
