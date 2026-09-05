package main

import (
	"strings"
	"testing"
)

// These tests run the CLI against the committed examples/ fixtures so the
// documented examples can never silently rot.

func TestFleetPlanExample(t *testing.T) {
	code, out, errout := runCap("fleet", "plan",
		"--inventory", examplesDir+"fleet-inventory.json",
		"--policy", examplesDir+"fleet-policy.json",
		"--image", "ecw230v3-282.bin",
		"--image-sha256", strings40(),
		"--family", "cloud",
		"--now", "2026-09-04T12:05:00Z")
	if code != 0 {
		t.Fatalf("example fleet plan failed: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "eligible=2 blocked=1 canary=1") {
		t.Fatalf("example inventory should yield 2 eligible / 1 blocked / 1 canary: %q", out)
	}
	if !strings.Contains(out, "ap-03-blocked    blocked") {
		t.Fatalf("ap-03 should be blocked (probable confidence + no backup): %q", out)
	}
}
