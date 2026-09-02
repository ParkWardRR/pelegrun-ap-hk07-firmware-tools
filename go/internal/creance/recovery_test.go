package creance

import "testing"

func TestScriptGatesBeforeSave(t *testing.T) {
	s := Script()
	// there must be a gated inspect step BEFORE `env save`
	saveIdx, gateIdx := -1, -1
	for i, c := range s {
		if c.Line == "env save" {
			saveIdx = i
		}
		if c.Gate && gateIdx == -1 {
			gateIdx = i
		}
	}
	if gateIdx == -1 || saveIdx == -1 || gateIdx >= saveIdx {
		t.Fatalf("must inspect (gate) before env save: gate=%d save=%d", gateIdx, saveIdx)
	}
	// env default -a must come before env save
	def := -1
	for i, c := range s {
		if c.Line == "env default -a" {
			def = i
		}
	}
	if def == -1 || def >= saveIdx {
		t.Fatal("env default -a must precede env save")
	}
}

func TestDecideSafeOnCompleteDefaults(t *testing.T) {
	ok, reason := Decide(ParseUboot("bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\n"))
	if !ok {
		t.Fatalf("complete defaults should be safe: %s", reason)
	}
}

func TestDecideStopsOnIncomplete(t *testing.T) {
	ok, reason := Decide(ParseUboot("bootcmd=bootipq\n")) // missing active_fw/app_part/rootfsname
	if ok || reason == "" {
		t.Fatal("incomplete defaults must STOP")
	}
}

func TestDecideStopsOnEmpty(t *testing.T) {
	if ok, _ := Decide(ParseUboot("")); ok {
		t.Fatal("empty defaults must STOP (possible vendor u-boot)")
	}
}
