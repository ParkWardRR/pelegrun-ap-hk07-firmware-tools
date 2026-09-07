package eyas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Recorded-fixture tests: each file under testdata/ is a representative device
// web response. The family is encoded in the filename prefix so adding a new
// captured page is a one-file change — drop it in, name it, done.
var fixtureFamily = map[string]Family{
	"cloud":   Cloud,
	"ews":     EwsLuCI,
	"fit":     Fit,
	"openwrt": OpenWrt,
	"unknown": Unknown,
}

func familyFor(t *testing.T, name string) Family {
	t.Helper()
	prefix := name
	if i := strings.IndexByte(name, '-'); i >= 0 {
		prefix = name[:i]
	}
	fam, ok := fixtureFamily[prefix]
	if !ok {
		t.Fatalf("fixture %q: unknown family prefix %q (want one of %v)", name, prefix, keys(fixtureFamily))
	}
	return fam
}

func TestClassifyFromRecordedFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/*.html")
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for _, f := range files {
		f := f
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			want := familyFor(t, name)
			if got := Classify(string(body)); got != want {
				t.Errorf("Classify(%s) = %v, want %v", name, got, want)
			}
		})
	}
}

// The full Fingerprint path, served over httptest from the same fixtures: the
// recognizable ones must resolve regardless of whether the marker is on "/" or
// the LuCI fallback path.
func TestFingerprintFromRecordedFixtures(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.html")
	for _, f := range files {
		f := f
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		want := familyFor(t, name)
		if want == Unknown {
			continue // Fingerprint returning Unknown is covered by Classify test
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		// LuCI markers live on the /cgi-bin/luci page in the real world.
		servePath := "/"
		if want == EwsLuCI || want == OpenWrt {
			servePath = "/cgi-bin/luci"
		}
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == servePath {
					w.Write(body)
					return
				}
				w.Write([]byte("<html>no markers</html>"))
			}))
			defer srv.Close()
			fam, err := Fingerprint(context.Background(), srv.Client(), srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			if fam != want {
				t.Errorf("Fingerprint(%s) = %v, want %v", name, fam, want)
			}
		})
	}
}

func keys(m map[string]Family) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
