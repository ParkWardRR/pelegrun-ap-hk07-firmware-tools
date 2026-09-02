package jess

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH reaches the EWS firmware's exec channel on the non-standard port 8822.
// These APs only offer legacy ssh-rsa (SHA-1) host keys, so the config must allow
// them explicitly — the reason a default client "connection refused"/handshake-fails.
type SSH struct {
	Host string // host:port; defaults to :8822 if no port
	User string // "root"
	Pass string // = the web-admin password
}

// ClientConfig builds an ssh.ClientConfig that accepts the legacy ssh-rsa host key
// algorithm these devices use. Exposed so it can be unit-tested without a device.
func (s SSH) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            s.User,
		Auth:            []ssh.AuthMethod{ssh.Password(s.Pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // first-contact recovery on a LAN device
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA, // ssh-rsa (SHA-1) = the one they need
		},
		Timeout: 10 * time.Second,
	}
}

func (s SSH) addr() string {
	if _, _, err := net.SplitHostPort(s.Host); err != nil {
		return net.JoinHostPort(s.Host, "8822")
	}
	return s.Host
}

// Run opens a session, runs one command via the exec channel (uid=0), returns stdout.
func (s SSH) Run(ctx context.Context, cmd string) (string, error) {
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
