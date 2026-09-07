package jess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// UploadImage stages a firmware image on the cloud firmware and returns the
// device-reported size + checksum from the local_upgrade_image validator. This
// only STAGES — nothing is flashed until FwUpgrade.
//
// The image's Senao header product_id is validated against the RUNNING
// firmware's own identity, not the target — e.g. flashing an EWS377-FIT image
// onto a device currently running ECW230v3 cloud firmware requires the image
// re-headed to product_id 284 (ECW230v3), not 300 (FIT). Once booted, the new
// firmware reports its own real identity; the header value only gates this
// upload. Use `quarry rehead <img> <img> --to <id>` before calling this.
func (c *Cloud) UploadImage(ctx context.Context, filename string, data []byte) (size int, checksum string, err error) {
	if err := c.DropCaches(ctx); err != nil {
		return 0, "", fmt.Errorf("drop_caches: %w", err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return 0, "", err
	}
	if _, err = fw.Write(data); err != nil {
		return 0, "", err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/cgi-bin/upload.cgi", &body)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Expect", "") // disable 100-continue
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, "", err
	}
	resp.Body.Close()

	// validate staged image
	raw, err := c.do(ctx, http.MethodGet, "/api/mgm/local_upgrade_image", nil)
	if err != nil {
		return 0, "", fmt.Errorf("validate staged image: %w", err)
	}
	var v struct {
		Size     int    `json:"image_size"`
		Checksum string `json:"image_checksum"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, "", err
	}
	if v.Size == 0 {
		return 0, "", fmt.Errorf("device rejected the image (size 0)")
	}
	return v.Size, v.Checksum, nil
}

// FwUpgrade triggers the LOCAL flash of the staged image (mode Upgrade_locally);
// a bare upgrade is an OTA no-op, so the mode is explicit.
func (c *Cloud) FwUpgrade(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPost, "/api/mgm/fw_upgrade", map[string]string{"mode": "Upgrade_locally"})
	return err
}

// Flashops performs the EWS LuCI two-step firmware flash: upload the image, then
// POST step=2 to confirm. keep=false does a clean cross-firmware flash. Requires a
// prior Login() (stok set).
func (l *LuCI) Flashops(ctx context.Context, filename string, data []byte, keep bool) error {
	if l.Stok == "" {
		return fmt.Errorf("flashops: login first (no stok)")
	}
	base := l.Base + "/cgi-bin/luci/;stok=" + l.Stok + "/admin/system/flashops"

	// step 1: multipart upload (field "image")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("image", filename)
	if _, err := fw.Write(data); err != nil {
		return err
	}
	mw.Close()
	up, err := http.NewRequestWithContext(ctx, http.MethodPost, base, &body)
	if err != nil {
		return err
	}
	up.Header.Set("Content-Type", mw.FormDataContentType())
	up.Header.Set("Expect", "")
	r1, err := l.Client.Do(up)
	if err != nil {
		return err
	}
	r1.Body.Close()

	// step 2: confirm
	keepv := ""
	if keep {
		keepv = "1"
	}
	form := url.Values{"step": {"2"}, "keep": {keepv}}
	cf, err := http.NewRequestWithContext(ctx, http.MethodPost, base, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	cf.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2, err := l.Client.Do(cf)
	if err != nil {
		return err
	}
	r2.Body.Close()
	return nil
}
