package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCrossflashCheckReportsIdentity(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
		case "/api/sys/sys_info":
			w.Write([]byte(`{"status_code":200,"data":{"firmware_version":"1.8.114.1","serial_number":"EPC1X420000000000000","mac_address":"88:DC:97:04:44:07"}}`))
		default:
			w.Write([]byte(`{"status_code":404}`))
		}
	}))
	defer srv.Close()
	ap := strings.TrimPrefix(srv.URL, "https://")

	code, out, errout := runCap("crossflash", "check", "--ap", ap)
	if code != 0 {
		t.Fatalf("check: code=%d err=%q", code, errout)
	}
	for _, want := range []string{"1.8.114.1", "EPC1X420000000000000", "88:DC:97:04:44:07", "282", "300", "284"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestCrossflashUsage(t *testing.T) {
	if code, out, _ := runCap("crossflash"); code != 0 || !strings.Contains(out, "pelegrun crossflash") {
		t.Fatalf("crossflash usage: code=%d out=%q", code, out)
	}
	if code, _, errout := runCap("crossflash", "bogus"); code != 1 || !strings.Contains(errout, "unknown subcommand") {
		t.Fatalf("crossflash bogus: code=%d err=%q", code, errout)
	}
}

func TestCrossflashCheckRequiresAP(t *testing.T) {
	if code, _, errout := runCap("crossflash", "check"); code != 1 || !strings.Contains(errout, "--ap") {
		t.Fatalf("missing --ap: code=%d err=%q", code, errout)
	}
}

func TestCrossflashPushRequiresImage(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
	}))
	defer srv.Close()
	ap := strings.TrimPrefix(srv.URL, "https://")
	if code, _, errout := runCap("crossflash", "push", "--ap", ap); code != 1 || !strings.Contains(errout, "--image") {
		t.Fatalf("missing --image: code=%d err=%q", code, errout)
	}
}

func TestCrossflashPushStagesAndGatesOnYes(t *testing.T) {
	var staged bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
		case "/api/mgm/drop_caches":
			w.Write([]byte(`{"status_code":200,"data":null}`))
		case "/cgi-bin/upload.cgi":
			staged = true
			w.Write([]byte(`{"status_code":200,"data":{}}`))
		case "/api/mgm/local_upgrade_image":
			if !staged {
				w.Write([]byte(`{"status_code":200,"data":{"image_size":0,"image_checksum":""}}`))
				return
			}
			w.Write([]byte(`{"status_code":200,"data":{"image_size":3,"image_checksum":"abc"}}`))
		case "/api/mgm/fw_upgrade":
			w.Write([]byte(`{"status_code":200,"data":null}`))
		default:
			w.Write([]byte(`{"status_code":404}`))
		}
	}))
	defer srv.Close()
	ap := strings.TrimPrefix(srv.URL, "https://")

	dir := t.TempDir()
	imgPath := dir + "/img.bin"
	if err := os.WriteFile(imgPath, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errout := runCap("crossflash", "push", "--ap", ap, "--image", imgPath)
	if code != 0 {
		t.Fatalf("push without --yes should succeed (stage only): code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "staged on") || strings.Contains(out, "fw_upgrade triggered") {
		t.Fatalf("expected stage-only output, got %q", out)
	}

	code, out, errout = runCap("crossflash", "push", "--ap", ap, "--image", imgPath, "--yes")
	if code != 0 {
		t.Fatalf("push --yes should trigger flash: code=%d err=%q", code, errout)
	}
	if !strings.Contains(out, "fw_upgrade triggered") {
		t.Fatalf("expected fw_upgrade to be triggered, got %q", out)
	}
}

func TestCrossflashPushRefusesSizeMismatch(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
		case "/api/mgm/drop_caches":
			w.Write([]byte(`{"status_code":200,"data":null}`))
		case "/cgi-bin/upload.cgi":
			w.Write([]byte(`{"status_code":200,"data":{}}`))
		case "/api/mgm/local_upgrade_image":
			// device reports a different size than what we sent
			w.Write([]byte(`{"status_code":200,"data":{"image_size":999,"image_checksum":"abc"}}`))
		default:
			w.Write([]byte(`{"status_code":404}`))
		}
	}))
	defer srv.Close()
	ap := strings.TrimPrefix(srv.URL, "https://")

	dir := t.TempDir()
	imgPath := dir + "/img.bin"
	os.WriteFile(imgPath, []byte("abc"), 0o644)

	code, _, errout := runCap("crossflash", "push", "--ap", ap, "--image", imgPath)
	if code == 0 || !strings.Contains(errout, "does not match") {
		t.Fatalf("size mismatch should be refused: code=%d err=%q", code, errout)
	}
}
