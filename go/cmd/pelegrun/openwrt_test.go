package main

import (
	"os"
	"strings"
	"testing"
)

func TestOpenWrtCheckDefaultRegistryIsEligible(t *testing.T) {
	code, out, errout := runCap("openwrt", "check")
	if code != 0 {
		t.Fatalf("default registry should be flash-ready: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "ELIGIBLE") {
		t.Fatalf("expected ELIGIBLE: %q", out)
	}
}

func TestOpenWrtPlanWritesFixedPartitionNotAB(t *testing.T) {
	dir := t.TempDir()
	envPath := dir + "/printenv.txt"
	if err := os.WriteFile(envPath, []byte("bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errout := runCap("openwrt", "plan", "--env", envPath, "--image", "factory.ubi")
	if code != 0 {
		t.Fatalf("plan should succeed with a complete env: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "FIXED rootfs partition") {
		t.Fatalf("expected the fixed-partition step: %q", out)
	}
	if strings.Contains(out, "reset-button hold reverts") {
		t.Fatalf("openwrt plan must never claim A/B rollback: %q", out)
	}
	if !strings.Contains(out, "GATE") {
		t.Fatalf("expected at least one [GATE] step marker: %q", out)
	}
}

func TestOpenWrtPlanRequiresEnvAndImage(t *testing.T) {
	if code, _, errout := runCap("openwrt", "plan"); code != 1 || !strings.Contains(errout, "--env") {
		t.Fatalf("missing --env: code=%d err=%q", code, errout)
	}
	dir := t.TempDir()
	envPath := dir + "/printenv.txt"
	os.WriteFile(envPath, []byte("active_fw=0\n"), 0o644)
	if code, _, errout := runCap("openwrt", "plan", "--env", envPath); code != 1 || !strings.Contains(errout, "--image") {
		t.Fatalf("missing --image: code=%d err=%q", code, errout)
	}
}

func TestOpenWrtPlanRefusesIncompleteEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := dir + "/printenv.txt"
	if err := os.WriteFile(envPath, []byte("ethaddr=x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errout := runCap("openwrt", "plan", "--env", envPath, "--image", "factory.ubi")
	if code == 0 || !strings.Contains(errout, "missing") {
		t.Fatalf("incomplete env should be refused: code=%d err=%q", code, errout)
	}
}

func TestOpenWrtUsage(t *testing.T) {
	if code, out, _ := runCap("openwrt"); code != 0 || !strings.Contains(out, "pelegrun openwrt") {
		t.Fatalf("openwrt usage: code=%d out=%q", code, out)
	}
	if code, _, errout := runCap("openwrt", "bogus"); code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("openwrt bogus: code=%d err=%q", code, errout)
	}
}
