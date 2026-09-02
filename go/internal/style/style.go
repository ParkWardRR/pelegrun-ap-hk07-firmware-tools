// Package style is tiny dependency-free ANSI styling for readable CLI output.
package style

import "os"

var on = os.Getenv("NO_COLOR") == ""

func wrap(code, s string) string {
	if !on {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func Bold(s string) string   { return wrap("1", s) }
func Dim(s string) string    { return wrap("2", s) }
func Red(s string) string    { return wrap("31", s) }
func Green(s string) string  { return wrap("32", s) }
func Yellow(s string) string { return wrap("33", s) }
func Cyan(s string) string   { return wrap("36", s) }
func Amber(s string) string  { return wrap("38;5;214", s) }

func OK(s string) string   { return Green("✔ ") + s }
func Warn(s string) string { return Yellow("▲ ") + s }
func No(s string) string   { return Red("✘ ") + s }
