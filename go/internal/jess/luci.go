package jess

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var stokRe = regexp.MustCompile(`stok=([a-f0-9]+)`)

// LuCI talks to the EWS firmware's LuCI web UI.
type LuCI struct {
	Base   string
	Client *http.Client
	Stok   string
}

// NewLuCI builds a LuCI adapter with a cookie jar and insecure client.
func NewLuCI(base string) *LuCI {
	c := InsecureClient(15 * time.Second)
	jar, _ := cookiejar.New(nil)
	c.Jar = jar
	// don't auto-follow the post-login redirect; we read the stok from it
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &LuCI{Base: strings.TrimRight(base, "/"), Client: c}
}

// Login posts username + md5(pw+\n) and captures the session stok token.
func (l *LuCI) Login(ctx context.Context, user, pass string) error {
	form := url.Values{"username": {user}, "password": {LuciPassword(pass)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.Base+"/cgi-bin/luci",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := l.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if m := stokRe.FindStringSubmatch(resp.Header.Get("Location")); m != nil {
		l.Stok = m[1]
		return nil
	}
	return http.ErrNoLocation
}
