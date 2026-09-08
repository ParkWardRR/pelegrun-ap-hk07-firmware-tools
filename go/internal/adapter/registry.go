package adapter

import (
	"encoding/json"
	"os"
	"sort"
)

// RegistrySchemaVersion is the on-disk version of a model registry.
const RegistrySchemaVersion = 1

// Registry is the machine-readable model DB (P12d): the published, actionable
// support state for every board the tool knows about. It stores one Support
// record per adapter and can validate them all at once.
type Registry struct {
	SchemaVersion int       `json:"schema_version"`
	Adapters      []Support `json:"adapters"`
}

// LoadRegistry reads a registry JSON (a missing file yields the builtin default).
func LoadRegistry(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultRegistry(), nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.SchemaVersion == 0 {
		r.SchemaVersion = RegistrySchemaVersion
	}
	return &r, nil
}

// Save writes the registry JSON.
func (r *Registry) Save(path string) error {
	if r.SchemaVersion == 0 {
		r.SchemaVersion = RegistrySchemaVersion
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Get returns the adapter record for a model (case-insensitive), or nil.
func (r *Registry) Get(model string) *Support {
	for i := range r.Adapters {
		if equalFold(r.Adapters[i].Model, model) {
			return &r.Adapters[i]
		}
	}
	return nil
}

// Validate runs Support.Validate on every adapter, returning a map of model ->
// problems (models with no problems are omitted). An empty map means the whole
// registry is internally consistent.
func (r *Registry) Validate() map[string][]error {
	out := map[string][]error{}
	for _, a := range r.Adapters {
		if errs := a.Validate(); len(errs) > 0 {
			out[a.Model] = errs
		}
	}
	return out
}

// Flashable returns the models permitted to run a destructive flash (CanFlash
// passes), sorted.
func (r *Registry) Flashable() []string {
	var models []string
	for _, a := range r.Adapters {
		if a.CanFlash() == nil {
			models = append(models, a.Model)
		}
	}
	sort.Strings(models)
	return models
}

// DefaultRegistry is the builtin model DB. It carries **two** records for the
// physical ap-hk07 board, one per firmware target — the registry is keyed on
// Model, and "which firmware is this write plan for" changes the recovery
// story enough (A/B rollback vs. a single fixed partition with no live
// fallback) that a single shared record would have to either lie about one
// of them or blur flash_ab/flash_fixed_partition together. Rather than that,
// the OpenWrt target gets its own Model string ("ap-hk07-openwrt") so
// Registry.Get never has to arbitrate between two different recovery
// promises for the same key.
//
// ap-hk07 (FIT) is `experimental`: the tool implements every safe primitive
// for it (fingerprint/inspect/access/backup/provision/A-B flash/UART+TFTP
// recovery/evidence), but it is NOT yet promoted to `verified` — that
// requires passing the hardware qualification matrix in the ROADMAP on real
// units. health_check is not implemented yet (false).
//
// ap-hk07-openwrt is also `experimental`, on the strength of the real-hardware
// evidence below — a persistent, reboot-surviving NAND install, not just a
// RAM-boot smoke test. It declares flash_fixed_partition (not flash_ab): see
// flash.FixedPartitionTarget's doc comment for why OpenWrt can't use the A/B
// path at all, and CapFlashFixedPartition's doc comment for why that's a
// distinct, honestly-weaker promise than flash_ab.
func DefaultRegistry() *Registry {
	return &Registry{
		SchemaVersion: RegistrySchemaVersion,
		Adapters: []Support{
			{
				Adapter: "ap-hk07",
				Model:   "ap-hk07",
				Tier:    TierExperimental,
				Capabilities: map[Capability]bool{
					CapFingerprint:  true,  // eyas
					CapInspect:      true,  // quarry/eyas
					CapAccess:       true,  // jess ssh/cloud/luci
					CapBackup:       true,  // mews + dump
					CapProvision:    true,  // band + hood
					CapFlashAB:      true,  // flash (inactive slot)
					CapUARTRecovery: true,  // creance
					CapTFTPRecovery: true,  // lure
					CapHealthCheck:  false, // not implemented yet
					CapEvidence:     true,  // mews + redact
				},
				Evidence: []string{
					"6 product ids verified across ~26 real images (quarry real_images test)",
					"Go<->Rust Code27 parity test",
					"lure TFTP unit + integration tests",
				},
			},
			{
				Adapter:       "ap-hk07-openwrt",
				Model:         "ap-hk07-openwrt",
				BoardRevision: "EWS377AP v3 (IPQ8072A)",
				Tier:          TierExperimental,
				Capabilities: map[Capability]bool{
					CapFingerprint:         true,  // eyas (OpenWrt family + BoardID shell probe)
					CapInspect:             true,  // quarry verify-ubi + eyas.BoardID
					CapAccess:              true,  // jess ssh (:22, no vendor auth)
					CapBackup:              true,  // mews + dump (mtd7/8/11 + full rootfs)
					CapProvision:           false, // no serial/append-only story on OpenWrt yet
					CapFlashAB:             false, // NOT A/B-capable — see flash_fixed_partition
					CapFlashFixedPartition: true,  // flash.PlanOpenWrt (rootfs @0x1000000, always)
					CapUARTRecovery:        true,  // manual tftpboot+nand write, hardware-proven
					CapTFTPRecovery:        true,  // same TFTP path used for both install and recovery
					CapHealthCheck:         false, // not implemented yet
					CapEvidence:            true,  // openwrt-ews377ap-v3/results-2026-09-06/ + mews
				},
				Evidence: []string{
					"first persistent NAND boot on real EWS377AP v3 hardware (not RAM-boot): " +
						"openwrt-ews377ap-v3/results-2026-09-06/STAGE3-SLOT0-INSTALL-RESULTS.md",
					"ethernet + dual-band WiFi (WPA2) + NSS offload confirmed working from NAND",
					"config (UCI + a written test file) survives a real reboot",
					"root-mount-requires-slot-0 and FIT config@<board>-name failure modes both " +
						"found and fixed by direct hardware debugging in the same results/ directory",
					"HTTP-only cross-flash (no UART) confirmed viable as a MECHANISM on this " +
						"board family via the OEM cloud updater (see `pelegrun crossflash`) — but " +
						"not yet proven for an OpenWrt payload specifically: on one unit, the same " +
						"HTTP path hit an identical NAND ECC error on the spare slot for two " +
						"different community UBI layouts (kernel-only and full 3-volume), while " +
						"genuine EnGenius FIT and ECW230v3 images wrote clean on the same slot — " +
						"CapUARTRecovery is not just the fallback here, it is the only path with a " +
						"demonstrated clean write for this specific payload.",
				},
			},
		},
	}
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
