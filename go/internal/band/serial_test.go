package band

import (
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

