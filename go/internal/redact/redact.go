// Package redact scrubs secrets and sensitive identifiers out of text before it
// leaves the operator's machine — support bundles, recorded fixtures, and logs.
// swallow handles credentials, session tokens, Wi-Fi keys, and device identity,
// and the ROADMAP makes redaction a first-class requirement ("fixture provenance
// and redaction policy"; "keep credentials out of plans, fixtures, logs, and
// support bundles"). This package is pure and deterministic: same input, same
// scrubbed output.
//
// It works on text (fw_printenv dumps, JSON API replies, shell logs, ssh config).
// Two mechanisms: category regexes for structural secrets (key=value / JSON
// secret pairs, bearer tokens, private-key blocks, MACs), and an explicit Values
// list for literals you already know are sensitive (serials, MACs, hostnames
// pulled from the inventory). Redaction fails safe by matching broadly; verify on
// real fixtures that nothing sensitive survives before publishing.
package redact

import (
	"regexp"
	"sort"
	"strings"
)

// Category is a class of secret the redactor can scrub.
type Category string

const (
	// CatSecretKV redacts the VALUE of a key=value / JSON pair whose key names a
	// secret (password, psk, token, secret, api_key, session/stok/sysauth, …).
	CatSecretKV Category = "secret-kv"
	// CatBearer redacts HTTP bearer tokens and bare JWTs.
	CatBearer Category = "bearer"
	// CatPrivateKey redacts PEM private-key blocks.
	CatPrivateKey Category = "private-key"
	// CatMAC redacts MAC addresses (identity — opt-in; off in Default).
	CatMAC Category = "mac"
)

// placeholder is what a redacted span becomes.
func placeholder(c Category) string { return "[REDACTED:" + string(c) + "]" }

var (
	// key is a secret key name; value is captured in the final group.
	secretKV = regexp.MustCompile(`(?i)("?\b(?:password|passwd|psk|wpa_psk|wifi_?key|wifi_?password|secret|api_?key|token|access_token|refresh_token|stok|sysauth|sessionid|private_key)\b"?\s*[:=]\s*"?)([^"\s,;}&]+)`)
	bearer   = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]+`)
	jwt      = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}`)
	pemKey   = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	macAddr  = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{2}[:-]){5}[0-9a-f]{2}\b`)
)

// Redactor scrubs text per its enabled categories plus any explicit literals.
type Redactor struct {
	Categories map[Category]bool
	Values     []string // exact literals to scrub (serials, MACs, hostnames)
}

// Default scrubs structural secrets (kv secrets, bearer/JWT, private keys) but
// NOT MAC addresses — those are identity and are redacted only on request.
func Default() *Redactor {
	return &Redactor{Categories: map[Category]bool{
		CatSecretKV:   true,
		CatBearer:     true,
		CatPrivateKey: true,
	}}
}

// All enables every category, including MAC redaction.
func All() *Redactor {
	r := Default()
	r.Categories[CatMAC] = true
	return r
}

// WithValues returns a copy of r with additional explicit literals to scrub.
func (r *Redactor) WithValues(values ...string) *Redactor {
	cp := &Redactor{Categories: map[Category]bool{}, Values: append([]string(nil), r.Values...)}
	for k, v := range r.Categories {
		cp.Categories[k] = v
	}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			cp.Values = append(cp.Values, v)
		}
	}
	return cp
}

// on reports whether category c is enabled.
func (r *Redactor) on(c Category) bool { return r.Categories != nil && r.Categories[c] }

// Text returns s with every enabled category and explicit value redacted.
// Private-key blocks are handled first (they can contain base64 that looks like
// other patterns), then bearer/JWT, then key=value secrets, then MACs, then the
// explicit literals (longest first, so a serial that contains a shorter value is
// scrubbed whole).
func (r *Redactor) Text(s string) string {
	if r.on(CatPrivateKey) {
		s = pemKey.ReplaceAllString(s, placeholder(CatPrivateKey))
	}
	if r.on(CatBearer) {
		s = bearer.ReplaceAllString(s, "Bearer "+placeholder(CatBearer))
		s = jwt.ReplaceAllString(s, placeholder(CatBearer))
	}
	if r.on(CatSecretKV) {
		s = secretKV.ReplaceAllString(s, "${1}"+placeholder(CatSecretKV))
	}
	if r.on(CatMAC) {
		s = macAddr.ReplaceAllString(s, placeholder(CatMAC))
	}
	if len(r.Values) > 0 {
		vals := append([]string(nil), r.Values...)
		sort.Slice(vals, func(i, j int) bool { return len(vals[i]) > len(vals[j]) })
		for _, v := range vals {
			s = replaceFold(s, v, "[REDACTED:value]")
		}
	}
	return s
}

// replaceFold replaces every case-insensitive occurrence of old in s.
func replaceFold(s, old, new string) string {
	if old == "" {
		return s
	}
	var b strings.Builder
	lowerS, lowerOld := strings.ToLower(s), strings.ToLower(old)
	for {
		i := strings.Index(lowerS, lowerOld)
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		b.WriteString(new)
		s = s[i+len(old):]
		lowerS = lowerS[i+len(old):]
	}
	return b.String()
}
