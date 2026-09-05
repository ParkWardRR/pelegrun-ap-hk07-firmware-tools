package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFleetFixtures drops an inventory and policy JSON into dir and returns
// their paths. The inventory has one eligible device (a) and one blocked (b, low
// identity confidence), both last-seen at the fixed --now used by the tests.
func writeFleetFixtures(t *testing.T, dir string) (inv, pol string) {
	t.Helper()
	inv = filepath.Join(dir, "inv.json")
	pol = filepath.Join(dir, "policy.json")
	invJSON := `{
      "schema_version": 1,
      "devices": [
        {
          "fleet_id": "a",
          "physical_identity": {"model": "ap-hk07", "hardware_serial": "SWLWX420001Q", "identity_confidence": "verified"},
          "software_identity": {"firmware_family": "cloud", "active_slot": "rootfs", "inactive_slot": "rootfs_1"},
          "access": {"management_address": "192.168.1.10", "last_seen_at": "2026-09-04T12:00:00Z"},
          "evidence": {"backup_bundle_sha256": "abc"},
          "eligibility": {"state": "unknown"}
        },
        {
          "fleet_id": "b",
          "physical_identity": {"model": "ap-hk07", "hardware_serial": "SWLWX420002N", "identity_confidence": "probable"},
          "software_identity": {"firmware_family": "cloud", "active_slot": "rootfs", "inactive_slot": "rootfs_1"},
          "access": {"management_address": "192.168.1.11", "last_seen_at": "2026-09-04T12:00:00Z"},
          "evidence": {"backup_bundle_sha256": "def"},
          "eligibility": {"state": "unknown"}
        }
      ]
    }`
	polJSON := `{
      "version": "v1",
      "supported_models": ["ap-hk07"],
      "allowed_families": ["cloud", "ews-luci"],
      "require_backup": true,
      "stale_after_seconds": 600,
      "canary": {"count": 1},
      "error_budget": {"max_failed": 0, "max_unknown": 0}
    }`
	if err := os.WriteFile(inv, []byte(invJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pol, []byte(polJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return inv, pol
}

func TestFleetPlanAndApply(t *testing.T) {
	dir := t.TempDir()
	inv, pol := writeFleetFixtures(t, dir)
	plan := filepath.Join(dir, "plan.json")
	now := "2026-09-04T12:01:00Z"

	code, out, errout := runCap("fleet", "plan",
		"--inventory", inv, "--policy", pol,
		"--image", "ecw230v3-282.bin", "--image-sha256", strings.Repeat("d", 64),
		"--family", "cloud", "--now", now, "--out", plan)
	if code != 0 {
		t.Fatalf("plan failed: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "eligible=1 blocked=1 canary=1") {
		t.Fatalf("plan summary wrong: %q", out)
	}
	if !strings.Contains(out, "✓ a") || !strings.Contains(out, "✗ b") {
		t.Fatalf("plan per-device lines wrong: %q", out)
	}
	if _, err := os.Stat(plan); err != nil {
		t.Fatalf("plan file not written: %v", err)
	}

	// Apply the fresh plan against the same inputs: guard should pass.
	code, out, errout = runCap("fleet", "apply",
		"--inventory", inv, "--policy", pol, "--plan", plan,
		"--image", "ecw230v3-282.bin", "--image-sha256", strings.Repeat("d", 64),
		"--family", "cloud", "--now", now, "--max-age", "10m")
	if code != 0 {
		t.Fatalf("apply failed: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "guard PASSED") || !strings.Contains(out, "no device was changed") {
		t.Fatalf("apply output wrong: %q", out)
	}
}

func TestFleetApplyRefusesChangedTarget(t *testing.T) {
	dir := t.TempDir()
	inv, pol := writeFleetFixtures(t, dir)
	plan := filepath.Join(dir, "plan.json")
	now := "2026-09-04T12:01:00Z"

	runCap("fleet", "plan", "--inventory", inv, "--policy", pol,
		"--image", "img.bin", "--image-sha256", strings.Repeat("d", 64),
		"--family", "cloud", "--now", now, "--out", plan)

	// Apply with a different image hash -> refused.
	code, _, errout := runCap("fleet", "apply",
		"--inventory", inv, "--policy", pol, "--plan", plan,
		"--image", "img.bin", "--image-sha256", strings.Repeat("e", 64),
		"--family", "cloud", "--now", now)
	if code != 1 || !strings.Contains(errout, "apply refused") {
		t.Fatalf("expected refusal on changed target: code=%d err=%q", code, errout)
	}
}

func TestFleetUsage(t *testing.T) {
	if code, out, _ := runCap("fleet"); code != 0 || !strings.Contains(out, "pelegrun fleet") {
		t.Fatalf("fleet usage: code=%d out=%q", code, out)
	}
	if code, _, errout := runCap("fleet", "bogus"); code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("fleet bogus: code=%d err=%q", code, errout)
	}
}
