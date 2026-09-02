package band

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestCheckCharKnown(t *testing.T) {
	c, err := CheckChar("EPC1X420001")
	if err != nil || c != '1' {
		t.Fatalf("got %q err %v (want '1')", c, err)
	}
}

func TestSerialRoundtrip(t *testing.T) {
	s, err := MakeSerial("EPC1", "X42", "0001")
	if err != nil || s != "EPC1X4200011" {
		t.Fatalf("got %q err %v", s, err)
	}
	if !ValidateSerial(s) {
		t.Fatal("should validate")
	}
	if ValidateSerial("EPC1X4200012") {
		t.Fatal("bad check char should fail")
	}
	mc, _ := ModelCode(s)
	if mc != "X42" {
		t.Fatalf("model code %q", mc)
	}
}

func TestSnextra(t *testing.T) {
	x, err := MakeSnextra("EPC1", "X42")
	if err != nil || x != "EPC1X420000000000000" || len(x) != SnextraLen {
		t.Fatalf("got %q err %v", x, err)
	}
}

func TestInventoryCollision(t *testing.T) {
	inv := &Inventory{}
	if err := inv.Reserve(Asset{Name: "ap1", Serial: "EPC1X4200011", MAC: "88:DC:97:00:00:01"}); err != nil {
		t.Fatal(err)
	}
	// same serial, different asset -> collision
	if err := inv.Reserve(Asset{Name: "ap2", Serial: "EPC1X4200011", MAC: "88:DC:97:00:00:02"}); !errors.Is(err, ErrCollision) {
		t.Fatalf("want ErrCollision, got %v", err)
	}
	// same MAC, different asset -> collision
	if err := inv.Reserve(Asset{Name: "ap3", Serial: "EPC1X4200025", MAC: "88:DC:97:00:00:01"}); !errors.Is(err, ErrCollision) {
		t.Fatalf("want ErrCollision (mac), got %v", err)
	}
	// unique -> ok
	if err := inv.Reserve(Asset{Name: "ap4", Serial: "EPC1X4200025", MAC: "88:DC:97:00:00:04"}); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryPersist(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inv.json")
	inv := &Inventory{}
	_ = inv.Reserve(Asset{Name: "ap1", Serial: "EPC1X4200011"})
	if err := inv.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil || len(got.Assets) != 1 {
		t.Fatalf("reload failed: %v %+v", err, got)
	}
}
