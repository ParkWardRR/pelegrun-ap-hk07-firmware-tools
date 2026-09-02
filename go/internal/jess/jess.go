// Package jess is the device-access tether: adapters that reach an ap-hk07 AP over
// its cloud JSON API, its LuCI web UI, or the SSH exec channel on port 8822, behind
// small focused types. Jesses tether the bird without harming it — these adapters
// read/act but never bypass the hood safety gate for env writes.
package jess

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"time"
)

// InsecureClient returns an http.Client that accepts the self-signed certs the
// cloud/FIT firmware and LuCI-over-HTTPS present (the firmware itself uses -k).
func InsecureClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // device self-signed
		},
	}
}

// LuciPassword hashes a web-admin password the way the EWS LuCI login expects:
// md5(password + "\n"), hex-encoded.
func LuciPassword(plain string) string {
	sum := md5.Sum([]byte(plain + "\n")) //nolint:gosec // vendor scheme, not for security
	return hex.EncodeToString(sum[:])
}
