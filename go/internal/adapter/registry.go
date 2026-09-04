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

// DefaultRegistry is the builtin model DB. ap-hk07 is `experimental`: the tool
// implements every safe primitive for it (fingerprint/inspect/access/backup/
// provision/A-B flash/UART+TFTP recovery/evidence), but it is NOT yet promoted to
// `verified` — that requires passing the hardware qualification matrix in the
// ROADMAP on real units. health_check is not implemented yet (false).
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
