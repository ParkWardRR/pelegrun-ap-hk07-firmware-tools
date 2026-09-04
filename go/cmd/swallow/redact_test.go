package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactFromFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bundle.txt")
	os.WriteFile(f, []byte("password=hunter2\nbootcmd=bootipq\nethaddr=88:DC:97:04:44:07\n"), 0o644)

	code, out, errout := runCap("redact", f)
	if code != 0 {
		t.Fatalf("redact failed: %d %q", code, errout)
	}
	if strings.Contains(out, "hunter2") {
		t.Fatalf("password leaked: %q", out)
	}
	if !strings.Contains(out, "bootcmd=bootipq") {
		t.Fatalf("non-secret should survive: %q", out)
	}
	// MAC preserved by default.
	if !strings.Contains(out, "88:DC:97:04:44:07") {
		t.Fatalf("MAC should survive without --mac: %q", out)
	}
}

func TestRedactMACAndValue(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bundle.txt")
	os.WriteFile(f, []byte("ethaddr=88:DC:97:04:44:07 serial=SWLWX420001Q\n"), 0o644)

	code, out, _ := runCap("redact", f, "--mac", "--value", "SWLWX420001Q")
	if code != 0 {
		t.Fatalf("redact failed: %d", code)
	}
	if strings.Contains(out, "88:DC:97:04:44:07") || strings.Contains(out, "SWLWX420001Q") {
		t.Fatalf("--mac and --value should scrub both: %q", out)
	}
}
