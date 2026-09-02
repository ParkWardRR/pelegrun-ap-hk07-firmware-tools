package band

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Asset is one provisioned device's immutable identity record.
type Asset struct {
	Name    string `json:"name"`
	Serial  string `json:"serial"` // synthetic snextra or 12-char serial
	MAC     string `json:"mac"`    // real label MAC
	Model   string `json:"model"`  // e.g. ECW230v3
	AddedAt string `json:"added_at"`
}

// Inventory is a collision-preventing local record of provisioned identities.
type Inventory struct {
	Assets []Asset `json:"assets"`
}

// ErrCollision is returned when a serial or MAC already belongs to another asset.
var ErrCollision = errors.New("serial/MAC already provisioned to another asset")

// Load reads an inventory JSON (missing file = empty inventory).
func Load(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Inventory{}, nil
	}
	if err != nil {
		return nil, err
	}
	var inv Inventory
	if err := json.Unmarshal(b, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// Save writes the inventory JSON.
func (inv *Inventory) Save(path string) error {
	b, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Reserve adds a, failing closed if its serial or MAC already exists under a
// different asset name. Re-reserving the same (name,serial,mac) is idempotent.
func (inv *Inventory) Reserve(a Asset) error {
	for _, x := range inv.Assets {
		serialHit := strings.EqualFold(x.Serial, a.Serial) && a.Serial != ""
		macHit := strings.EqualFold(x.MAC, a.MAC) && a.MAC != ""
		if (serialHit || macHit) && !strings.EqualFold(x.Name, a.Name) {
			return fmt.Errorf("%w: %q vs existing %q", ErrCollision, a.Name, x.Name)
		}
	}
	inv.Assets = append(inv.Assets, a)
	return nil
}
