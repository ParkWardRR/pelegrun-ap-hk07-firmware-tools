package dump

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ValidationStatus tracks how far a dump has been proven.
type ValidationStatus string

const (
	StatusPlanned  ValidationStatus = "planned"  // commands planned, nothing captured
	StatusCaptured ValidationStatus = "captured" // artifacts written + hashed on device
	StatusVerified ValidationStatus = "verified" // on-device hashes matched the pulled copy
)

// PartitionRecord is one partition's manifest entry.
type PartitionRecord struct {
	Index    int    `json:"index"`
	Dev      string `json:"dev"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Size     int64  `json:"size_bytes"`
	Artifact string `json:"artifact"`
	SHA256   string `json:"sha256,omitempty"` // filled after capture
	Critical bool   `json:"critical"`
}

// Manifest is the dump's sidecar: device identity, the flash map, per-artifact
// hashes, and how far the capture has been validated. It is the provenance record
// that makes an on-device backup trustworthy and rediscoverable.
type Manifest struct {
	SchemaVersion  int               `json:"schema_version"`
	Model          string            `json:"model,omitempty"`
	Serial         string            `json:"serial,omitempty"`
	DestDir        string            `json:"dest_dir"`
	CapturedAt     string            `json:"captured_at,omitempty"` // RFC3339, set by accessor
	Tool           string            `json:"tool"`
	NAND           bool              `json:"nand"`
	EstimatedBytes int64             `json:"estimated_bytes"`
	Partitions     []PartitionRecord `json:"partitions"`
	Validation     ValidationStatus  `json:"validation"`
}

// ManifestSchemaVersion is the on-disk version for a dump manifest.
const ManifestSchemaVersion = 1

// NewManifest builds a planned manifest for the given layout. SHA256/CapturedAt
// are filled by the accessor after it runs the plan; validation starts "planned".
func NewManifest(parts []Partition, dest, tool string, opts PlanOptions) *Manifest {
	m := &Manifest{
		SchemaVersion:  ManifestSchemaVersion,
		Model:          opts.Model,
		Serial:         opts.Serial,
		DestDir:        strings.TrimRight(dest, "/"),
		Tool:           tool,
		NAND:           opts.NAND,
		EstimatedBytes: EstimatedBytes(parts),
		Validation:     StatusPlanned,
	}
	for _, p := range parts {
		m.Partitions = append(m.Partitions, PartitionRecord{
			Index:    p.Index,
			Dev:      p.Dev,
			Name:     p.Name,
			Role:     Classify(p.Name).String(),
			Size:     p.Size,
			Artifact: Artifact(p),
			Critical: p.Critical(),
		})
	}
	return m
}

// Save writes the manifest JSON.
func (m *Manifest) Save(path string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadManifest reads a manifest JSON.
func LoadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Verify proves the dump is intact: every artifact's on-device hash must match
// the hash of the copy pulled to the host (keys are artifact names, values are
// lowercase hex SHA-256). A missing or mismatched artifact is an error. On
// success the manifest's per-partition SHA256 is recorded and validation advances
// to "verified". "Prove, don't assume."
func (m *Manifest) Verify(onDevice, pulled map[string]string) error {
	var problems []string
	for i := range m.Partitions {
		name := m.Partitions[i].Artifact
		dev, okD := onDevice[name]
		host, okH := pulled[name]
		switch {
		case !okD:
			problems = append(problems, name+": no on-device hash")
		case !okH:
			problems = append(problems, name+": not pulled to host")
		case !strings.EqualFold(dev, host):
			problems = append(problems, fmt.Sprintf("%s: on-device %s != host %s", name, short(dev), short(host)))
		default:
			m.Partitions[i].SHA256 = strings.ToLower(dev)
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		return fmt.Errorf("dump verification failed: %s", strings.Join(problems, "; "))
	}
	m.Validation = StatusVerified
	return nil
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
