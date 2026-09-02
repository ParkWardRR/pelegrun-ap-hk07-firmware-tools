package jess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestLuciPassword(t *testing.T) {
	// md5("admin\n")
	if got := LuciPassword("admin"); got != "456b7016a916a4b178dd72b947c152b7" {
		t.Fatalf("LuciPassword(admin) = %s", got)
	}
}

func TestCloudLoginAndSysInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"abc.def.ghi"}}`))
		case "/api/sys/sys_info":
			if r.Header.Get("Authorization") != "Bearer abc.def.ghi" {
				w.Write([]byte(`{"status_code":401}`))
				return
			}
			w.Write([]byte(`{"status_code":200,"data":{"firmware_version":"1.8.112.13","serial_number":"224203612","mac_address":"88:DC:97:04:44:07"}}`))
		default:
			w.Write([]byte(`{"status_code":404}`))
		}
	}))
	defer srv.Close()

	c := NewCloud(srv.URL)
	c.Client = srv.Client()
	if err := c.Login(context.Background(), "admin", "admin"); err != nil {
		t.Fatal(err)
	}
	si, err := c.SysInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if si.Serial != "224203612" || si.MAC != "88:DC:97:04:44:07" || si.Firmware != "1.8.112.13" {
		t.Fatalf("sysinfo = %+v", si)
	}
}

func TestLuciLoginStok(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/luci" && r.Method == http.MethodPost {
			r.ParseForm()
			if r.PostForm.Get("password") != LuciPassword("admin") {
				w.WriteHeader(403)
				return
			}
			w.Header().Set("Location", "/cgi-bin/luci/;stok=deadbeefcafe1234/admin/status/overview/")
			w.WriteHeader(302)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	l := NewLuCI(srv.URL)
	l.Client = srv.Client()
	l.Client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if err := l.Login(context.Background(), "admin", "admin"); err != nil {
		t.Fatal(err)
	}
	if l.Stok != "deadbeefcafe1234" {
		t.Fatalf("stok = %q", l.Stok)
	}
}

func TestSSHConfigAllowsLegacyRSA(t *testing.T) {
	cfg := SSH{User: "root", Pass: "admin"}.ClientConfig()
	found := false
	for _, a := range cfg.HostKeyAlgorithms {
		if a == ssh.KeyAlgoRSA { // "ssh-rsa"
			found = true
		}
	}
	if !found {
		t.Fatalf("config must allow legacy ssh-rsa; got %v", cfg.HostKeyAlgorithms)
	}
	if cfg.User != "root" {
		t.Fatal("user")
	}
	_ = strings.TrimSpace // keep import tidy across edits
}

func TestSSHDefaultPort(t *testing.T) {
	if a := (SSH{Host: "10.0.0.1"}).addr(); a != "10.0.0.1:8822" {
		t.Fatalf("default port = %s", a)
	}
	if a := (SSH{Host: "10.0.0.1:22"}).addr(); a != "10.0.0.1:22" {
		t.Fatalf("explicit port = %s", a)
	}
}
