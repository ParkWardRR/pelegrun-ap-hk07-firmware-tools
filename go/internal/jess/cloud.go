package jess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Cloud talks to the unadopted cloud/FIT firmware's local JSON API.
type Cloud struct {
	Base   string // e.g. http://172.16.1.68
	Client *http.Client
	token  string
}

// NewCloud builds a Cloud adapter with an insecure client (self-signed device).
func NewCloud(base string) *Cloud {
	return &Cloud{Base: strings.TrimRight(base, "/"), Client: InsecureClient(15 * time.Second)}
}

type apiResp struct {
	StatusCode int             `json:"status_code"`
	Data       json.RawMessage `json:"data"`
}

func (c *Cloud) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ar apiResp
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if ar.StatusCode != 200 {
		return nil, fmt.Errorf("cloud %s %s: status %d", method, path, ar.StatusCode)
	}
	return ar.Data, nil
}

// Login authenticates (default admin/admin) and stores the bearer token.
func (c *Cloud) Login(ctx context.Context, user, pass string) error {
	data, err := c.do(ctx, http.MethodPost, "/api/sys/login", map[string]string{"username": user, "password": pass})
	if err != nil {
		return err
	}
	var t struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	if t.Token == "" {
		return fmt.Errorf("login: empty token")
	}
	c.token = t.Token
	return nil
}

// SysInfo is the subset of /api/sys/sys_info we care about.
type SysInfo struct {
	Firmware string `json:"firmware_version"`
	Serial   string `json:"serial_number"`
	MAC      string `json:"mac_address"`
}

// SysInfo reads identity + firmware from the device.
func (c *Cloud) SysInfo(ctx context.Context) (SysInfo, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/sys/sys_info", nil)
	var s SysInfo
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(data, &s)
}

// SetForceAC points the AP at a controller (highest-priority discovery override).
func (c *Cloud) SetForceAC(ctx context.Context, ip string) error {
	_, err := c.do(ctx, http.MethodPost, "/api/mgm/force_ac", map[string]string{"ip": ip})
	return err
}

// DropCaches clears any previously-staged upgrade image. The real admin GUI
// calls this immediately before every upload.cgi POST (confirmed by decoding
// its JS bundle); UploadImage calls it for the same reason — matching the
// genuine device flow exactly, not a guessed subset of it.
func (c *Cloud) DropCaches(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPost, "/api/mgm/drop_caches", nil)
	return err
}
