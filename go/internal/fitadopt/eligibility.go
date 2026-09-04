// Package fitadopt is the P10a read-only eligibility gate for supported FIT
// real-serial adoption. It decides whether a device/image/serial combination is
// an ALLOWED adoption operation before any write or reboot — it performs no I/O
// and makes no device changes. The real-serial constraint is absolute: adoption
// preserves the device's existing serial; this package never generates, reuses,
// or spoofs one (ROADMAP P10, "real serial only; no spoofing").
package fitadopt

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/band"
)

// MinFitVersion is the supported floor for real-serial FIT adoption.
const MinFitVersion = "1.1.65"

// hex64 matches a lowercase/uppercase 64-char SHA-256 hex digest.
var hex64 = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// Request is a proposed FIT adoption, assembled read-only from device inspection
// (eyas), the operator's target image, and the device's own reported serial.
type Request struct {
	Model           string   // device model, e.g. ap-hk07
	FirmwareFamily  string   // must be "fit" (eyas family string)
	FitVersion      string   // dotted FIT version the target adopts to
	ImageSHA256     string   // target image digest (64 hex)
	ImageProvenance string   // where the image came from (non-empty, auditable)
	RealSerial      string   // serial READ FROM the device — never generated
	BootloaderState string   // observed bootloader/boot state
	RecoveryRoutes  []string // e.g. ["ab-rollback","uart","tftp"] — at least one
}

// Result is the eligibility outcome. Reasons list every block so an operator sees
// the whole picture, not just the first failure.
type Result struct {
	Eligible bool
	Reasons  []string
}

// Validate applies every FIT adoption precondition. It is deterministic and
// side-effect free; a true result means the operation is *allowed to be planned*,
// not that it succeeded — post-adoption proof (P10b) is a separate gate.
func Validate(r Request) Result {
	var reasons []string

	if !strings.EqualFold(r.FirmwareFamily, "fit") {
		reasons = append(reasons, fmt.Sprintf("firmware family %q is not FIT", r.FirmwareFamily))
	}
	if r.Model == "" {
		reasons = append(reasons, "device model is unknown; refuse adoption without a model/layout match")
	}

	switch cmp, err := CompareVersions(r.FitVersion, MinFitVersion); {
	case err != nil:
		reasons = append(reasons, fmt.Sprintf("FIT version %q is unparseable", r.FitVersion))
	case cmp < 0:
		reasons = append(reasons, fmt.Sprintf("FIT version %s is below the supported floor %s", r.FitVersion, MinFitVersion))
	}

	if !hex64.MatchString(r.ImageSHA256) {
		reasons = append(reasons, "target image SHA-256 is missing or not a 64-char hex digest")
	}
	if strings.TrimSpace(r.ImageProvenance) == "" {
		reasons = append(reasons, "target image has no recorded provenance/source")
	}

	if err := validateRealSerial(r.RealSerial); err != nil {
		reasons = append(reasons, err.Error())
	}

	if len(r.RecoveryRoutes) == 0 {
		reasons = append(reasons, "no recovery route established (need at least one of ab-rollback/uart/tftp)")
	}

	sort.Strings(reasons)
	return Result{Eligible: len(reasons) == 0, Reasons: reasons}
}

// validateRealSerial enforces that the serial looks like a device-read identity,
// not a placeholder or a generated value. A 12-char serial must pass the Code27
// check; anything shorter/blank or an obvious placeholder is refused.
func validateRealSerial(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("no real serial read from the device (adoption must preserve the existing serial)")
	}
	lower := strings.ToLower(s)
	for _, bad := range []string{"spoof", "fake", "generated", "placeholder", "000000000000"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("serial %q looks generated/placeholder; only a real device serial is allowed", s)
		}
	}
	if len(s) == band.SerialLen && !band.ValidateSerial(s) {
		return fmt.Errorf("serial %q fails the Code27 check character", s)
	}
	return nil
}

// CompareVersions compares two dotted numeric versions (e.g. "1.1.65"). It
// returns -1, 0, or 1, or an error if either side has a non-numeric component.
// Missing trailing components are treated as zero (1.1 == 1.1.0).
func CompareVersions(a, b string) (int, error) {
	pa, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

func parseVersion(v string) ([]int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("empty version")
	}
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("non-numeric version component %q", p)
		}
		out[i] = n
	}
	return out, nil
}
