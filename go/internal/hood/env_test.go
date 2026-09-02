package hood

import "testing"

const good = `bootcmd=bootipq
active_fw=0
app_part=0
rootfsname=rootfs
ethaddr=00:11:22:33:44:55
snextra=EPC1X420000000000000
`

func TestParseAndComplete(t *testing.T) {
	e := ParsePrintenv(good)
	if !e.IsComplete() {
		t.Fatalf("expected complete, missing %v", e.Missing())
	}
	if e["bootcmd"] != "bootipq" {
		t.Fatalf("bootcmd = %q", e["bootcmd"])
	}
}

func TestIncompleteMissing(t *testing.T) {
	e := ParsePrintenv("bootcmd=bootipq\nethaddr=x\n")
	if e.IsComplete() {
		t.Fatal("should be incomplete")
	}
	got := e.Missing()
	// active_fw, app_part, rootfsname missing
	if len(got) != 3 {
		t.Fatalf("missing = %v", got)
	}
}

func TestPlanSetAppendOnly(t *testing.T) {
	e := ParsePrintenv(good)
	cmd, err := e.PlanSet("snextra", "EPC1X420000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "fw_setenv snextra EPC1X420000000000000" {
		t.Fatalf("cmd = %q", cmd)
	}
}

func TestPlanSetRefusesEmptyValue(t *testing.T) {
	e := ParsePrintenv(good)
	if _, err := e.PlanSet("snextra", ""); err != ErrEmptyValue {
		t.Fatalf("want ErrEmptyValue, got %v", err) // empty value = delete = brick vector
	}
}

func TestPlanSetRefusesIncompleteEnv(t *testing.T) {
	e := ParsePrintenv("ethaddr=x\n")
	if _, err := e.PlanSet("snextra", "y"); err == nil {
		t.Fatal("must refuse writes on an incomplete env")
	}
}

func TestVerifyAfterSet(t *testing.T) {
	e := ParsePrintenv(good)
	if err := VerifyAfterSet(e, "snextra", "EPC1X420000000000000"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyAfterSet(e, "snextra", "WRONG"); err == nil {
		t.Fatal("should fail on mismatch")
	}
}
