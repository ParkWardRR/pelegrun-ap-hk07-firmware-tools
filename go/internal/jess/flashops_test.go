package jess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudUploadValidateFwUpgrade(t *testing.T) {
	var droppedCaches, uploaded, upgraded bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sys/login":
			w.Write([]byte(`{"status_code":200,"data":{"token":"t"}}`))
		case "/api/mgm/drop_caches":
			droppedCaches = true
			w.Write([]byte(`{"status_code":200,"data":null}`))
		case "/cgi-bin/upload.cgi":
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				if _, _, e := r.FormFile("file"); e == nil {
					uploaded = true
				}
			}
			w.Write([]byte(`{"status_code":200,"data":{}}`))
		case "/api/mgm/local_upgrade_image":
			if !uploaded {
				w.Write([]byte(`{"status_code":200,"data":{"image_size":0,"image_checksum":""}}`))
				return
			}
			w.Write([]byte(`{"status_code":200,"data":{"image_size":28185362,"image_checksum":"abc"}}`))
		case "/api/mgm/fw_upgrade":
			var b map[string]string
			json_decode(r, &b)
			if b["mode"] == "Upgrade_locally" {
				upgraded = true
			}
			w.Write([]byte(`{"status_code":200,"data":null}`))
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
	size, sum, err := c.UploadImage(context.Background(), "ecw230v3-282.bin", []byte("firmwarebytes"))
	if err != nil || size != 28185362 || sum != "abc" {
		t.Fatalf("upload: size=%d sum=%s err=%v", size, sum, err)
	}
	if err := c.FwUpgrade(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !upgraded {
		t.Fatal("fw_upgrade must send mode=Upgrade_locally")
	}
	if !droppedCaches {
		t.Fatal("UploadImage must call drop_caches first, matching the real GUI's request sequence")
	}
}

func TestLuciFlashopsTwoStep(t *testing.T) {
	var gotUpload, gotStep2 bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "" && r.ContentLength > 0 && r.PostFormValue("step") == "" {
				if r.ParseMultipartForm(1<<20) == nil {
					if _, _, e := r.FormFile("image"); e == nil {
						gotUpload = true
					}
				}
			}
			if r.PostFormValue("step") == "2" {
				gotStep2 = true
			}
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	l := NewLuCI(srv.URL)
	l.Client = srv.Client()
	l.Stok = "deadbeef"
	if err := l.Flashops(context.Background(), "ecw230v3-282.bin", []byte("fw"), false); err != nil {
		t.Fatal(err)
	}
	if !gotUpload || !gotStep2 {
		t.Fatalf("upload=%v step2=%v", gotUpload, gotStep2)
	}
}

func TestFlashopsRequiresStok(t *testing.T) {
	l := NewLuCI("http://x")
	if err := l.Flashops(context.Background(), "f", []byte("x"), false); err == nil {
		t.Fatal("must require stok (login) first")
	}
}

func TestCloudDropCaches(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/mgm/drop_caches" && r.Method == http.MethodPost {
			called = true
			w.Write([]byte(`{"status_code":200,"data":null}`))
			return
		}
		w.Write([]byte(`{"status_code":404}`))
	}))
	defer srv.Close()

	c := NewCloud(srv.URL)
	c.Client = srv.Client()
	if err := c.DropCaches(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("DropCaches must POST /api/mgm/drop_caches")
	}
}
