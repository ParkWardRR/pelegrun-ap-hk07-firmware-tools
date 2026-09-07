package main

import (
	"os"
	"testing"

	"strings"
)

// examples/ lives at the repo root: go/cmd/pelegrun -> ../../../examples.
const examplesDir = "../../../examples/"

func TestFitCheckExample(t *testing.T) {
	code, out, errout := runCap("fit", "check", "--request", examplesDir+"fit-request.json")
	if code != 0 {
		t.Fatalf("example fit request should be eligible: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "ELIGIBLE") {
		t.Fatalf("expected ELIGIBLE: %q", out)
	}
}

func TestFitProveExample(t *testing.T) {
	code, out, errout := runCap("fit", "prove",
		"--expected", examplesDir+"fit-expected.json",
		"--observed", examplesDir+"fit-observed.json")
	if code != 0 {
		t.Fatalf("example proof should pass: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "PROVEN") {
		t.Fatalf("expected PROVEN: %q", out)
	}
}

func TestFitCheckBadRequestFails(t *testing.T) {
	// A request with a below-floor FIT version is written inline via a temp file.
	dir := t.TempDir()
	bad := dir + "/bad.json"
	if err := os.WriteFile(bad, []byte(`{"model":"ap-hk07","firmware_family":"fit","fit_version":"1.1.29","image_sha256":"`+strings40()+`","image_provenance":"x","real_serial":"SWLWX420001T","recovery_routes":["uart"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCap("fit", "check", "--request", bad)
	if code == 0 || !strings.Contains(out, "NOT ELIGIBLE") {
		t.Fatalf("below-floor FIT should be rejected: code=%d out=%q", code, out)
	}
}

func TestFitUsage(t *testing.T) {
	if code, out, _ := runCap("fit"); code != 0 || !strings.Contains(out, "pelegrun fit") {
		t.Fatalf("fit usage: code=%d out=%q", code, out)
	}
	if code, _, errout := runCap("fit", "bogus"); code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("fit bogus: code=%d err=%q", code, errout)
	}
}

func strings40() string { return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }
