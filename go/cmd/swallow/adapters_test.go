package main

import (
	"strings"
	"testing"
)

func TestAdaptersListDefault(t *testing.T) {
	code, out, _ := runCap("adapters", "list")
	if code != 0 || !strings.Contains(out, "ap-hk07") || !strings.Contains(out, "tier=experimental") {
		t.Fatalf("adapters list: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "flash:YES") {
		t.Fatalf("ap-hk07 should be flashable: %q", out)
	}
}

func TestAdaptersValidateDefault(t *testing.T) {
	code, out, _ := runCap("adapters", "validate")
	if code != 0 || !strings.Contains(out, "registry OK") {
		t.Fatalf("default registry should validate: code=%d out=%q", code, out)
	}
}

func TestAdaptersUnknownSub(t *testing.T) {
	code, _, errout := runCap("adapters", "frobnicate")
	if code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("unknown adapters sub: code=%d err=%q", code, errout)
	}
}
