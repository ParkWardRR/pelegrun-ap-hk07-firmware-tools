package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const procMTDFixture = `dev:    size   erasesize  name
mtd7: 00080000 00020000 "0:APPSBLENV"
mtd8: 00140000 00020000 "0:APPSBL"
mtd11: 00080000 00020000 "0:ART"
`

func TestDumpPlanFromFile(t *testing.T) {
	dir := t.TempDir()
	pm := filepath.Join(dir, "proc-mtd.txt")
	if err := os.WriteFile(pm, []byte(procMTDFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	mf := filepath.Join(dir, "manifest.json")

	code, out, errout := runCap("dump", "plan", "--dest", "/tmp/pelegrun-dump",
		"--proc-mtd", pm, "--model", "ap-hk07", "--manifest", mf)
	if code != 0 {
		t.Fatalf("dump plan failed: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "3 partitions") {
		t.Fatalf("expected 3 partitions in summary: %q", out)
	}
	// The plan must be read-only.
	if strings.Contains(out, "of=/dev/mtd") {
		t.Fatal("dump plan must never write an mtd")
	}
	if !strings.Contains(out, "dd if=/dev/mtd11") {
		t.Fatalf("expected an ART dump command: %q", out)
	}
	if _, err := os.Stat(mf); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
}

func TestDumpPlanNAND(t *testing.T) {
	dir := t.TempDir()
	pm := filepath.Join(dir, "proc-mtd.txt")
	os.WriteFile(pm, []byte(procMTDFixture), 0o644)
	code, out, _ := runCap("dump", "plan", "--dest", "/tmp/d", "--proc-mtd", pm, "--nand")
	if code != 0 || !strings.Contains(out, "nanddump") {
		t.Fatalf("nand plan should use nanddump: code=%d out=%q", code, out)
	}
}

func TestDumpUnsafeDestWarns(t *testing.T) {
	dir := t.TempDir()
	pm := filepath.Join(dir, "proc-mtd.txt")
	os.WriteFile(pm, []byte(procMTDFixture), 0o644)
	code, out, _ := runCap("dump", "plan", "--dest", "/overlay/x", "--proc-mtd", pm)
	if code != 0 || !strings.Contains(out, "WARNING") {
		t.Fatalf("on-flash dest should warn: code=%d out=%q", code, out)
	}
}

func TestDumpRefusesDeviceDest(t *testing.T) {
	dir := t.TempDir()
	pm := filepath.Join(dir, "proc-mtd.txt")
	os.WriteFile(pm, []byte(procMTDFixture), 0o644)
	code, _, errout := runCap("dump", "plan", "--dest", "/dev/mtd11", "--proc-mtd", pm)
	if code != 1 || errout == "" {
		t.Fatalf("device dest should be refused: code=%d err=%q", code, errout)
	}
}

func TestDumpUsage(t *testing.T) {
	if code, out, _ := runCap("dump"); code != 0 || !strings.Contains(out, "pelegrun dump") {
		t.Fatalf("dump usage: code=%d out=%q", code, out)
	}
}
