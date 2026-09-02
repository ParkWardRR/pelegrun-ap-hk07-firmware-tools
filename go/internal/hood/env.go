// Package hood is the safety gate that keeps the raptor calm.
//
// It parses a u-boot environment, refuses to act on an incomplete one, and only
// ever plans APPEND-ONLY single-field writes. It has no function that can erase
// the env or save a partial one — the two things that brick a board (Constitution
// clause I). All logic here is pure; hood emits command strings for an accessor to
// run, and never does I/O itself.
package hood

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Env is a parsed u-boot environment (key -> value).
type Env map[string]string

// RequiredBootVars must all be present for the env to be considered bootable.
// Their absence is exactly what turns a valid-but-incomplete env into a brick.
var RequiredBootVars = []string{"bootcmd", "active_fw", "app_part", "rootfsname"}

var (
	// ErrIncomplete means the env is missing a required boot variable.
	ErrIncomplete = errors.New("env is incomplete; refusing to write (would risk a brick)")
	// ErrEmptyValue means a write had an empty value — in u-boot that DELETES the
	// variable, which violates the append-only invariant.
	ErrEmptyValue = errors.New("empty value would delete the variable; refused (append-only)")
	// ErrEmptyKey means a blank key was given.
	ErrEmptyKey = errors.New("empty key")
)

// ParsePrintenv parses the output of `fw_printenv` / u-boot `printenv`
// (one `key=value` per line; blank lines and comments ignored). Values may
// contain '='.
func ParsePrintenv(s string) Env {
	e := make(Env)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		e[strings.TrimSpace(line[:i])] = line[i+1:]
	}
	return e
}

// Missing returns the required boot variables absent from e, sorted.
func (e Env) Missing() []string {
	var m []string
	for _, k := range RequiredBootVars {
		if _, ok := e[k]; !ok {
			m = append(m, k)
		}
	}
	sort.Strings(m)
	return m
}

// IsComplete reports whether every RequiredBootVars is present.
func (e Env) IsComplete() bool { return len(e.Missing()) == 0 }

// PlanSet returns the single append-only command to set key=value, but ONLY if
// the current env is already complete and the value is non-empty. This is the one
// sanctioned way to mutate the env; there is deliberately no PlanErase / PlanReset.
func (e Env) PlanSet(key, value string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
	if value == "" {
		return "", ErrEmptyValue
	}
	if !e.IsComplete() {
		return "", fmt.Errorf("%w: missing %v — recover with `env default -a` over UART first", ErrIncomplete, e.Missing())
	}
	return fmt.Sprintf("fw_setenv %s %s", key, value), nil
}

// VerifyAfterSet checks that a re-read env reflects the intended write AND is
// still complete. "Prove, don't assume" (Constitution clause II).
func VerifyAfterSet(after Env, key, want string) error {
	if !after.IsComplete() {
		return fmt.Errorf("%w after write: missing %v", ErrIncomplete, after.Missing())
	}
	if got := after[key]; got != want {
		return fmt.Errorf("verify failed: %s = %q, wanted %q", key, got, want)
	}
	return nil
}
