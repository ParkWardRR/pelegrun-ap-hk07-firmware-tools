package jess

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestOpenWrtSSHConfigAllowsEd25519(t *testing.T) {
	cfg := OpenWrtSSH{User: "root"}.ClientConfig()
	found := false
	for _, a := range cfg.HostKeyAlgorithms {
		if a == ssh.KeyAlgoED25519 {
			found = true
		}
	}
	if !found {
		t.Fatalf("config must allow ed25519 (dropbear's common default); got %v", cfg.HostKeyAlgorithms)
	}
	if cfg.User != "root" {
		t.Fatal("user")
	}
}

func TestOpenWrtSSHDefaultPort22(t *testing.T) {
	if a := (OpenWrtSSH{Host: "10.0.0.1"}).addr(); a != "10.0.0.1:22" {
		t.Fatalf("default port = %s, want :22 (not :8822 — this is the mainline path)", a)
	}
	if a := (OpenWrtSSH{Host: "10.0.0.1:2222"}).addr(); a != "10.0.0.1:2222" {
		t.Fatalf("explicit port = %s", a)
	}
}

func TestOpenWrtSSHAllowsEmptyPasswordAuth(t *testing.T) {
	// OpenWrt commonly ships no root password at all; the zero-value Pass
	// must still produce a usable (if empty) password auth method rather
	// than requiring a caller to special-case "no password".
	cfg := OpenWrtSSH{User: "root", Pass: ""}.ClientConfig()
	if len(cfg.Auth) == 0 {
		t.Fatal("empty Pass should still yield a password auth method, not zero auth methods")
	}
}

func TestOpenWrtSSHAppendsCallerAuthKeys(t *testing.T) {
	extra := ssh.PublicKeys() // a real signer isn't needed to test wiring
	cfg := OpenWrtSSH{User: "root", AuthKeys: []ssh.AuthMethod{extra}}.ClientConfig()
	if len(cfg.Auth) < 2 {
		t.Fatalf("caller-supplied AuthKeys must be appended, got %d auth methods", len(cfg.Auth))
	}
}
