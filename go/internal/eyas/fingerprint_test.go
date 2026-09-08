package eyas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := map[string]Family{
		`<script src="/static/js/main.abc.chunk.js">`:           Cloud,
		`<script src="/luci-static/resources/md5.js"></script>`: EwsLuCI,
		`<input name="password_plain_text" type="password">`:    EwsLuCI,
		`action="/cgi-bin/luci"`:                                OpenWrt, // bare LuCI, no EWS vendor markers
		`<title>EWS377-FIT</title>`:                             Fit,
		`<html><body>nothing here</body></html>`:                Unknown,
	}
	for body, want := range cases {
		if got := Classify(body); got != want {
			t.Errorf("Classify(%.30q) = %v, want %v", body, got, want)
		}
	}
}

func TestFingerprintOverHTTP(t *testing.T) {
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte(`<div id="app"></div><script src="/static/js/main.f2.chunk.js"></script>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer cloud.Close()

	fam, err := Fingerprint(context.Background(), cloud.Client(), cloud.URL)
	if err != nil || fam != Cloud {
		t.Fatalf("cloud: got %v err %v", fam, err)
	}

	luci := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/luci" {
			w.Write([]byte(`<form action="/cgi-bin/luci"><input name="password_plain_text"></form>`))
			return
		}
		w.Write([]byte(`<html>redirecting</html>`)) // "/" gives nothing useful
	}))
	defer luci.Close()

	fam, err = Fingerprint(context.Background(), luci.Client(), luci.URL)
	if err != nil || fam != EwsLuCI {
		t.Fatalf("luci: got %v err %v", fam, err)
	}
}
