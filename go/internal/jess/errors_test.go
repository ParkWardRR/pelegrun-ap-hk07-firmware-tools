package jess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cloudSrv serves a fixed JSON body for any path (used to exercise error paths).
func cloudSrv(t *testing.T, body string) *Cloud {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c := NewCloud(srv.URL)
	c.Client = srv.Client()
	return c
}

func TestCloudNon200IsError(t *testing.T) {
	c := cloudSrv(t, `{"status_code":403}`)
	if err := c.Login(context.Background(), "admin", "admin"); err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Errorf("want status-403 error, got %v", err)
	}
}

func TestCloudLoginEmptyToken(t *testing.T) {
	c := cloudSrv(t, `{"status_code":200,"data":{"token":""}}`)
	if err := c.Login(context.Background(), "admin", "admin"); err == nil || !strings.Contains(err.Error(), "empty token") {
		t.Errorf("want empty-token error, got %v", err)
	}
}

func TestCloudSysInfoErrorPropagates(t *testing.T) {
	c := cloudSrv(t, `{"status_code":500}`)
	if _, err := c.SysInfo(context.Background()); err == nil {
		t.Error("SysInfo should surface the non-200 error")
	}
}

func TestCloudSetForceAC(t *testing.T) {
	var gotIP string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]string
		json_decode(r, &m)
		gotIP = m["ip"]
		w.Write([]byte(`{"status_code":200,"data":{}}`))
	}))
	defer srv.Close()
	c := NewCloud(srv.URL)
	c.Client = srv.Client()
	if err := c.SetForceAC(context.Background(), "172.16.6.111"); err != nil {
		t.Fatal(err)
	}
	if gotIP != "172.16.6.111" {
		t.Errorf("force_ac ip = %q, want 172.16.6.111", gotIP)
	}
}

func TestLuciLoginNoLocationIsError(t *testing.T) {
	// A login that doesn't redirect with a stok= Location must error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // no Location header
	}))
	defer srv.Close()
	l := NewLuCI(srv.URL)
	l.Client = srv.Client()
	l.Client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if err := l.Login(context.Background(), "admin", "pw"); err == nil {
		t.Error("expected an error when no stok is returned")
	}
}
