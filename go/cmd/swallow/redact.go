package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/redact"
)

// cmdRedact scrubs secrets from a support bundle / fixture / log before sharing.
// Reads a file (or stdin with "-"/omitted), writes the redacted text to stdout.
// Structural secrets (passwords, tokens, private keys) are always scrubbed;
// --mac also redacts MAC addresses; --value <literal> scrubs an exact string
// (repeatable — pass serials/hostnames you know are sensitive).
func cmdRedact(out io.Writer, a []string) error {
	var data []byte
	var err error
	src := firstPositional(a)
	if src == "" || src == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(src)
	}
	if err != nil {
		return err
	}

	r := redact.Default()
	if has(a, "--mac") {
		r = redact.All()
	}
	if vals := argVals(a, "--value"); len(vals) > 0 {
		r = r.WithValues(vals...)
	}

	fmt.Fprint(out, r.Text(string(data)))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Fprintln(out)
	}
	return nil
}

// firstPositional returns the first arg that is not a flag or a flag's value.
func firstPositional(a []string) string {
	skip := false
	for _, v := range a {
		if skip {
			skip = false
			continue
		}
		if v == "--value" { // consumes the next token
			skip = true
			continue
		}
		if len(v) > 0 && v[0] == '-' && v != "-" {
			continue
		}
		return v
	}
	return ""
}

// argVals collects every value following an occurrence of key (repeatable flag).
func argVals(a []string, key string) []string {
	var out []string
	for i, v := range a {
		if v == key && i+1 < len(a) {
			out = append(out, a[i+1])
		}
	}
	return out
}
