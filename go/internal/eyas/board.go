package eyas

import (
	"context"
	"errors"
	"strings"
)

// Runner executes a single shell command against the device and returns its
// stdout. Shaped to match fleetexec.Runner without importing that package —
// eyas stays a leaf (HTTP + shell probes only), never an execution-layer
// dependency.
type Runner func(ctx context.Context, cmd string) (string, error)

// ErrNoBoardID means neither `ubus call system board` nor the board.json
// fallback produced a board identifier — the device may not be OpenWrt, or
// the shell session lacks the expected tools.
var ErrNoBoardID = errors.New("eyas: no board identifier found (ubus/board.json both empty)")

// BoardID confirms an OpenWrt device's exact board identifier, e.g.
// "engenius,ews377ap-v3". The HTML fingerprint in Classify can tell you a
// device is *some* OpenWrt build; it cannot see /etc/board.json, so it can
// never confirm this is specifically the ap-hk07 board this project supports.
// That distinction only exists on-device, hence a shell probe rather than an
// HTTP one.
func BoardID(ctx context.Context, run Runner) (string, error) {
	// Preferred: ubus, present on any OpenWrt build with the base system.
	if out, err := run(ctx, `ubus call system board 2>/dev/null | sed -n 's/.*"board_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'`); err == nil {
		if name := strings.TrimSpace(out); name != "" {
			return name, nil
		}
	}
	// Fallback: read board.json directly (covers minimal images without ubus
	// fully wired, e.g. a RAM-boot initramfs).
	out, err := run(ctx, `cat /tmp/sysinfo/board_name 2>/dev/null; echo; sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /etc/board.json 2>/dev/null | head -1`)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			return name, nil
		}
	}
	return "", ErrNoBoardID
}
