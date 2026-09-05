package fitadopt

import (
	"strings"
	"testing"

	"github.com/ParkWardRR/pelegrun-ap-hk07-firmware-tools/internal/band"
)

func goodRequest() Request {
	serial, _ := band.MakeSerial("SWLW", "X42", "0001")
	return Request{
		Model:           "ap-hk07",
		FirmwareFamily:  "fit",
		FitVersion:      "1.1.65",
		ImageSHA256:     strings.Repeat("a", 64),
		ImageProvenance: "vendor portal, downloaded 2026-09-01",
		RealSerial:      serial,
		RecoveryRoutes:  []string{"ab-rollback", "uart"},
	}
}

func TestValidateEligible(t *testing.T) {
	if res := Validate(goodRequest()); !res.Eligible {
		t.Fatalf("expected eligible, blocked by %v", res.Reasons)
	}
}

func TestValidateBlocks(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Request)
	}{
		{"not fit family", func(r *Request) { r.FirmwareFamily = "cloud" }},
		{"below floor", func(r *Request) { r.FitVersion = "1.1.64" }},
		{"unparseable version", func(r *Request) { r.FitVersion = "1.x" }},
		{"bad image hash", func(r *Request) { r.ImageSHA256 = "short" }},
		{"no provenance", func(r *Request) { r.ImageProvenance = "  " }},
		{"empty serial", func(r *Request) { r.RealSerial = "" }},
		{"spoofed serial", func(r *Request) { r.RealSerial = "spoof-serial" }},
		{"bad check char", func(r *Request) { r.RealSerial = "SWLWX420001A" }}, // wrong check char
		{"no recovery route", func(r *Request) { r.RecoveryRoutes = nil }},
		{"no model", func(r *Request) { r.Model = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := goodRequest()
			tc.mut(&r)
			if res := Validate(r); res.Eligible {
				t.Fatalf("%s should block adoption", tc.name)
			}
		})
	}
}

func TestValidateAboveFloorOk(t *testing.T) {
	r := goodRequest()
	r.FitVersion = "1.2.0"
	if res := Validate(r); !res.Eligible {
		t.Fatalf("newer FIT should be eligible, blocked by %v", res.Reasons)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.1.65", "1.1.65", 0},
		{"1.1.64", "1.1.65", -1},
		{"1.2.0", "1.1.99", 1},
		{"1.1", "1.1.0", 0},
		{"2", "1.9.9", 1},
	}
	for _, tc := range cases {
		got, err := CompareVersions(tc.a, tc.b)
		if err != nil {
			t.Fatalf("compare(%s,%s): %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Fatalf("compare(%s,%s)=%d want %d", tc.a, tc.b, got, tc.want)
		}
	}
	if _, err := CompareVersions("1.a", "1.0"); err == nil {
		t.Fatal("non-numeric component should error")
	}
}
