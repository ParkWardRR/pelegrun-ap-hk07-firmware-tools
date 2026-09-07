package jess

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// OpenWrtSSH reaches a mainline/community OpenWrt device's dropbear on the
// standard port 22. Distinct from SSH (the EWS vendor fork on :8822): dropbear
// commonly ships an ed25519-only host key, and root frequently has no
// password at all (empty-string auth) or a key, not the EWS web-admin
// password scheme — reusing SSH's RSA-only, password-only ClientConfig would
// fail the handshake outright, the mirror image of the bug SSH's own doc
// comment describes for the legacy EWS side.
type OpenWrtSSH struct {
	Host     string          // host:port; defaults to :22 if no port
	User     string          // "root"
	Pass     string          // often empty — OpenWrt commonly ships no root password
	AuthKeys []ssh.AuthMethod // optional: e.g. ssh.PublicKeys(signer) for key auth
}

// ClientConfig builds an ssh.ClientConfig that accepts both ed25519 (dropbear's
// common default) and RSA host keys, and tries password auth (possibly empty)
// plus any caller-supplied key-based methods.
func (s OpenWrtSSH) ClientConfig() *ssh.ClientConfig {
	auth := append([]ssh.AuthMethod{ssh.Password(s.Pass)}, s.AuthKeys...)
	return &ssh.ClientConfig{
		User:            s.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // first-contact recovery on a LAN device
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA,
		},
		Timeout: 10 * time.Second,
	}
}

func (s OpenWrtSSH) addr() string {
	if _, _, err := net.SplitHostPort(s.Host); err != nil {
		return net.JoinHostPort(s.Host, "22")
	}
	return s.Host
}

// Run opens a session, runs one command via the exec channel (uid=0), returns
// stdout+stderr combined. Satisfies fleetexec.Runner's shape without either
// package importing the other.
func (s OpenWrtSSH) Run(ctx context.Context, cmd string) (string, error) {
	conn, err := ssh.Dial("tcp", s.addr(), s.ClientConfig())
	if err != nil {
		return "", fmt.Errorf("dial %s: %w", s.addr(), err)
	}
	defer conn.Close()
	sess, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}
