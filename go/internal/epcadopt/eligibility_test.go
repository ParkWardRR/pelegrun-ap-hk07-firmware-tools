package epcadopt

import "testing"

func goodRequest() Request {
	return Request{
		Model:          "ap-hk07",
		FirmwareFamily: "cloud",
		RealSerial:     "SWLWX420001T",
		RealMAC:        "AA:BB:CC:11:22:33",
		ControllerAddr: "controller.example",
		OrgID:          "org-000000000000000000000000",
		NetworkID:      "net-000000000000000000000000",
		RecoveryRoutes: []string{"uart"},
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
		{"no model", func(r *Request) { r.Model = "" }},
		{"no controller addr", func(r *Request) { r.ControllerAddr = "" }},
		{"no org", func(r *Request) { r.OrgID = "" }},
		{"no network", func(r *Request) { r.NetworkID = "" }},
		{"empty serial", func(r *Request) { r.RealSerial = "" }},
		{"empty mac", func(r *Request) { r.RealMAC = "" }},
		{"spoofed serial", func(r *Request) { r.RealSerial = "spoof-serial" }},
		{"placeholder mac", func(r *Request) { r.RealMAC = "00:00:00:00:00:00" }},
		{"no recovery route", func(r *Request) { r.RecoveryRoutes = nil }},
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
