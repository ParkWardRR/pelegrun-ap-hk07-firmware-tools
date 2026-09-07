package eyas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFamilyStringAndAccessHint(t *testing.T) {
	cases := []struct {
		f          Family
		str        string
		hintSubstr string
	}{
		{Cloud, "cloud", "no shell"},
		{EwsLuCI, "ews-luci", "8822"},
		{Fit, "fit", "no shell"},
		{OpenWrt, "openwrt", ":22"},
		{Unknown, "unknown", "unknown"},
	}
	for _, c := range cases {
		if got := c.f.String(); got != c.str {
			t.Errorf("String(%d) = %q, want %q", c.f, got, c.str)
		}
		if got := c.f.AccessHint(); !strings.Contains(got, c.hintSubstr) {
			t.Errorf("AccessHint(%s) = %q, want substring %q", c.str, got, c.hintSubstr)
		}
	}
}

// A mainline OpenWrt LuCI page (bare cgi-bin/luci, no EnGenius vendor
// markers) must classify as OpenWrt, not EwsLuCI — the latter would send an
// operator to SSH port 8822 with a web-admin password against a device whose
// dropbear is on :22 with no auth at all.
func TestClassifyBareLuCIIsOpenWrtNotEwsLuCI(t *testing.T) {
	body := `<html><body>
		<link rel="stylesheet" href="/luci-static/resources/cascade.css">
		<form action="/cgi-bin/luci/"><input name="luci_username"></form>
	</body></html>`
	if got := Classify(body); got != OpenWrt {
		t.Errorf("Classify(bare LuCI) = %v, want OpenWrt (bare cgi-bin/luci must not match EwsLuCI's vendor markers)", got)
	}
}

// The EWS vendor fork's *stronger* markers (md5.js, password_plain_text) must
// still win over the generic OpenWrt markers when both are present, since a
// real EWS page also happens to serve on /cgi-bin/luci.
func TestClassifyVendorMarkersStillWinEwsLuCI(t *testing.T) {
	body := `<script src="/luci-static/resources/md5.js"></script>
		<input name="password_plain_text"><form action="/cgi-bin/luci"></form>`
	if got := Classify(body); got != EwsLuCI {
		t.Errorf("Classify(vendor markers) = %v, want EwsLuCI", got)
	}
}

func TestFingerprintUnreachableReturnsUnknown(t *testing.T) {
	// A closed server: both probe paths error, so Fingerprint returns Unknown
	// with no error (best-effort discovery).
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // now nothing is listening

	fam, err := Fingerprint(context.Background(), &http.Client{Timeout: time.Second}, url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fam != Unknown {
		t.Errorf("got %v, want Unknown", fam)
	}
}

func TestFingerprintServerErrorStillClassifiesBody(t *testing.T) {
	// Even a 500 with a recognizable body should classify (we read the body).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<title>EWS377-FIT</title>`))
	}))
	defer srv.Close()

	fam, err := Fingerprint(context.Background(), srv.Client(), srv.URL)
	if err != nil || fam != Fit {
		t.Errorf("got %v err %v, want Fit", fam, err)
	}
}
