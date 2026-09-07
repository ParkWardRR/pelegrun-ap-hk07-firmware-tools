package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// apServer is a synthetic cloud-firmware AP: login + sys_info only, with an
// obviously-fake serial/MAC (never a real device identity, Constitution VII).
func apServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
		case "/api/sys/sys_info":
			w.Write([]byte(`{"status_code":200,"data":{"firmware_version":"1.8.114.1","serial_number":"SWLWX420001T","mac_address":"AA:BB:CC:11:22:33"}}`))
		default:
			w.Write([]byte(`{"status_code":404}`))
		}
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "https://")
}

func TestEpcCheckEligibleReportsIdentity(t *testing.T) {
	ap := apServer(t)
	code, out, errout := runCap("epc", "check", "--ap", ap,
		"--controller", "controller.example", "--org", "org-0", "--network", "net-0",
		"--recovery", "ab-rollback,uart")
	if code != 0 {
		t.Fatalf("check should be eligible: code=%d err=%q", code, errout)
	}
	for _, want := range []string{"SWLWX420001T", "AA:BB:CC:11:22:33", "ELIGIBLE", "NOT VERIFIED"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %q", want, out)
		}
	}
}

func TestEpcCheckRequiresController(t *testing.T) {
	if code, _, errout := runCap("epc", "check", "--ap", "x"); code != 1 || !strings.Contains(errout, "--controller") {
		t.Fatalf("missing --controller: code=%d err=%q", code, errout)
	}
}

func TestEpcCheckReportsIneligibility(t *testing.T) {
	ap := apServer(t)
	// No --recovery, no --org/--network → eligibility must fail (and exit 1).
	code, out, _ := runCap("epc", "check", "--ap", ap, "--controller", "controller.example")
	if code != 1 {
		t.Fatalf("expected exit 1 for ineligible: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "NOT ELIGIBLE") {
		t.Fatalf("expected NOT ELIGIBLE report: %q", out)
	}
}

func TestEpcPlanIsGatedAndMutatesNothing(t *testing.T) {
	ap := apServer(t)
	code, out, errout := runCap("epc", "plan", "--ap", ap,
		"--controller", "controller.example", "--org", "org-0", "--network", "net-0",
		"--recovery", "uart")
	if code != 0 {
		t.Fatalf("plan should succeed: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "[GATE]") {
		t.Fatalf("expected at least one [GATE] step: %q", out)
	}
	if !strings.Contains(out, "nothing mutated") {
		t.Fatalf("plan must state it mutates nothing: %q", out)
	}
}

func TestEpcPlanRefusesIneligible(t *testing.T) {
	ap := apServer(t)
	// Missing --org/--network/--recovery → epcadopt.Plan refuses.
	code, _, errout := runCap("epc", "plan", "--ap", ap, "--controller", "controller.example")
	if code != 1 || !strings.Contains(errout, "not eligible") {
		t.Fatalf("ineligible plan should be refused: code=%d err=%q", code, errout)
	}
}

func TestEpcProveHappyPath(t *testing.T) {
	dir := t.TempDir()
	exp := dir + "/exp.json"
	obs := dir + "/obs.json"
	os.WriteFile(exp, []byte(`{"model":"ap-hk07","real_serial":"SWLWX420001T","real_mac":"AA:BB:CC:11:22:33","controller_addr":"controller.example"}`), 0o644)
	os.WriteFile(obs, []byte(`{"model":"ap-hk07","real_serial":"SWLWX420001T","real_mac":"AA:BB:CC:11:22:33","controller_addr":"controller.example","controller_pointed":true,"reachable":true,"auth_ok":true,"inventory_match":true,"adopted":true,"survived_ap_reboot":true,"survived_ctrl_restart":true}`), 0o644)

	code, out, errout := runCap("epc", "prove", "--expected", exp, "--observed", obs)
	if code != 0 {
		t.Fatalf("prove should pass: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "PROVEN") {
		t.Fatalf("expected PROVEN: %q", out)
	}
}

func TestEpcProveReportsEveryFailure(t *testing.T) {
	dir := t.TempDir()
	exp := dir + "/exp.json"
	obs := dir + "/obs.json"
	os.WriteFile(exp, []byte(`{"model":"ap-hk07","real_serial":"SWLWX420001T","real_mac":"AA:BB:CC:11:22:33","controller_addr":"controller.example"}`), 0o644)
	// Serial changed AND durability checks failed — expect multiple failures.
	os.WriteFile(obs, []byte(`{"model":"ap-hk07","real_serial":"CHANGED0001","real_mac":"AA:BB:CC:11:22:33","controller_addr":"controller.example","controller_pointed":true,"reachable":true,"auth_ok":true,"inventory_match":true,"adopted":true,"survived_ap_reboot":false,"survived_ctrl_restart":false}`), 0o644)

	code, out, _ := runCap("epc", "prove", "--expected", exp, "--observed", obs)
	if code != 1 {
		t.Fatalf("expected proof failure exit 1: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "serial CHANGED") || !strings.Contains(out, "AP reboot") || !strings.Contains(out, "controller restart") {
		t.Fatalf("expected all failures reported: %q", out)
	}
}

func TestEpcUsage(t *testing.T) {
	if code, out, _ := runCap("epc"); code != 0 || !strings.Contains(out, "pelegrun epc") {
		t.Fatalf("epc usage: code=%d out=%q", code, out)
	}
	if code, _, errout := runCap("epc", "bogus"); code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("epc bogus: code=%d err=%q", code, errout)
	}
}
