// Package tui is the professional, falconry-themed Bubble Tea interface for
// swallow. It is a live dashboard: navigating the menu renders real output from
// the internal packages (eyas, band, hood, mews), so what you see is what the
// tool actually computes — including the safety refusals.
package tui

import (
	"fmt"
	"strings"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/band"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/eyas"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/mews"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	glyph, title, blurb string
	render              func() string
}

// Model is the root TUI model.
type Model struct {
	version string
	items   []item
	cursor  int
	w, h    int
}

// New builds the model with all screens wired to the real packages.
func New(version string) Model {
	return Model{
		version: version,
		items: []item{
			{"◆", "Overview", "the safety story", overview},
			{"◇", "Discover", "fingerprint firmware (eyas)", discover},
			{"✚", "Provision", "unique serials (band)", provision},
			{"⛨", "Env safety", "append-only gate (hood)", envSafety},
			{"❒", "Backup", "evidence bundle (mews)", backup},
			{"§", "Legal", "scope & license", legal},
		},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			m.cursor = len(m.items) - 1
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.w == 0 {
		m.w, m.h = 96, 30
	}
	header := m.header()

	// menu
	var menu strings.Builder
	for i, it := range m.items {
		line := fmt.Sprintf("%s %s", it.glyph, it.title)
		if i == m.cursor {
			menu.WriteString(itemSelected.Render(line))
			menu.WriteString("  " + dim.Render(it.blurb))
		} else {
			menu.WriteString(itemStyle.Render(line))
		}
		menu.WriteString("\n")
	}
	menuW := 26
	menuPanel := menuBox.Width(menuW).Height(m.bodyH()).Render(strings.TrimRight(menu.String(), "\n"))

	contentW := m.w - menuW - 6
	if contentW < 30 {
		contentW = 30
	}
	body := m.items[m.cursor].render()
	contentPanel := contentBox.Width(contentW).Height(m.bodyH()).Render(body)

	main := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, contentPanel)
	foot := footStyle.Render(dim.Render("↑/↓") + " move   " + dim.Render("g/G") + " top/bottom   " + dim.Render("q") + " quit    " + dim.Render("· unofficial · not affiliated with EnGenius/Senao ·"))
	return lipgloss.JoinVertical(lipgloss.Left, header, main, foot)
}

func (m Model) bodyH() int {
	h := m.h - 6
	if h < 12 {
		h = 12
	}
	return h
}

func (m Model) header() string {
	title := titleStyle.Render("🦅 swallow")
	tag := tagStyle.Render("  cross-flash & recover EnGenius/Senao ap-hk07 APs — without bricking them")
	ver := verStyle.Render("v" + m.version)
	badge := badgeStyle.Render("ap-hk07")
	left := title + tag
	right := badge + " " + ver
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return headerBox.Width(m.w - 2).Render(line)
}

// ---- content renderers (real package output) ----

func overview() string {
	var b strings.Builder
	b.WriteString(h2.Render("Why UART is usually unnecessary") + "\n\n")
	b.WriteString(okStyle.Render("1.") + " Writes go to the " + codeStyle.Render("INACTIVE") + " A/B slot — a bad image is undone with a reset-button hold.\n")
	b.WriteString(okStyle.Render("2.") + " The bootloader env is " + codeStyle.Render("APPEND-ONLY") + " — the tool cannot erase it or save a partial one.\n\n")
	b.WriteString(h2.Render("Recovery ladder") + "\n")
	rows := [][2]string{
		{"network flash", "no UART — dual A/B slot"},
		{"network env-repair", "no UART — append-only fw_setenv"},
		{"UART env-repair", "gated env default -a → inspect → env save"},
		{"UART TFTP re-flash", "truly dead board — lure calls it back"},
	}
	for i, r := range rows {
		b.WriteString(fmt.Sprintf("  %s  %-20s %s\n", dim.Render(fmt.Sprintf("%d.", i+1)), r[0], dim.Render(r[1])))
	}
	b.WriteString("\n" + dim.Render("A valid-but-incomplete env is the one thing that bricks. This tool can't make one."))
	return b.String()
}

func discover() string {
	var b strings.Builder
	b.WriteString(h2.Render("eyas · firmware fingerprint") + "\n\n")
	fams := []struct{ fam eyas.Family; marker string }{
		{eyas.Cloud, "static/js/main"},
		{eyas.EwsLuCI, "md5.js / password_plain_text / cgi-bin/luci"},
		{eyas.Fit, "ews377-fit / fitcontroller"},
	}
	for _, f := range fams {
		b.WriteString(fmt.Sprintf("  %s  %s\n", codeStyle.Render(fmt.Sprintf("%-9s", f.fam.String())), dim.Render("web marker: "+f.marker)))
		b.WriteString(fmt.Sprintf("            %s %s\n", dim.Render("access:"), f.fam.AccessHint()))
	}
	b.WriteString("\n" + dim.Render("live:  swallow discover http://<ap-ip>"))
	return b.String()
}

func provision() string {
	var b strings.Builder
	b.WriteString(h2.Render("band · unique serials (Code27)") + "\n\n")
	b.WriteString(dim.Render(fmt.Sprintf("  %-12s %-6s %-14s %-22s %s\n", "model", "code", "serial12", "snextra (field 19)", "ok")))
	models := [][2]string{{"ECW230v3", "X42"}, {"EWS377-FIT", "X45"}, {"EWS377AP v3", "X44"}}
	for i, mdl := range models {
		suffix := fmt.Sprintf("%04d", i+1)
		s, _ := band.MakeSerial("SWLW", mdl[1], suffix)
		x, _ := band.MakeSnextra("SWLW", mdl[1])
		ok := "✔"
		if !band.ValidateSerial(s) {
			ok = "✘"
		}
		b.WriteString(fmt.Sprintf("  %-12s %s %-14s %-22s %s\n", mdl[0], codeStyle.Render(fmt.Sprintf("%-6s", mdl[1])), s, dim.Render(x), okStyle.Render(ok)))
	}
	b.WriteString("\n" + dim.Render("each physical AP gets a different serial; collisions fail closed against the inventory."))
	return b.String()
}

func envSafety() string {
	good := "bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nsnextra=00000000000000000000\n"
	wiped := "ethaddr=00:03:7f:12:3e:87\n"
	var b strings.Builder
	b.WriteString(h2.Render("hood · append-only env gate") + "\n\n")

	e := hood.ParsePrintenv(good)
	b.WriteString(okStyle.Render("complete env") + dim.Render("  (bootcmd, active_fw, app_part, rootfsname present)") + "\n")
	cmd, _ := e.PlanSet("snextra", "SWLWX42000000000000A")
	b.WriteString("  " + okStyle.Render("allowed → ") + codeStyle.Render(cmd) + "\n\n")

	w := hood.ParsePrintenv(wiped)
	b.WriteString(noStyle.Render("wiped/incomplete env") + dim.Render(fmt.Sprintf("  (missing %v)", w.Missing())) + "\n")
	_, err := w.PlanSet("snextra", "SWLWX42000000000000A")
	b.WriteString("  " + noStyle.Render("REFUSED → ") + warnStyle.Render(oneline(err.Error())) + "\n\n")
	b.WriteString(dim.Render("no PlanErase / PlanReset exists. empty values (which delete vars) are also refused."))
	return b.String()
}

func backup() string {
	var b strings.Builder
	b.WriteString(h2.Render("mews · backup / evidence bundle") + dim.Render("  (hard gate before any flash)") + "\n\n")
	for _, a := range mews.Plan() {
		tag := dim.Render("optional")
		if a.Critical {
			tag = warnStyle.Render("critical")
		}
		b.WriteString(fmt.Sprintf("  %s  %-20s %s  %s\n", okStyle.Render("▢"), a.Name, dim.Render(a.Command), tag))
	}
	b.WriteString("\n" + dim.Render("mtd11 (ART = RF calibration + factory MACs) is read-only-backup, never written."))
	b.WriteString("\n" + dim.Render("bundle: "+mews.BundleName("2026-09-02", "ScuderiaToroRosso", "88:DC:97:04:44:07")))
	return b.String()
}

func legal() string {
	var b strings.Builder
	b.WriteString(h2.Render("Scope & license") + "\n\n")
	b.WriteString(dim.Render("Unofficial — ") + "not affiliated with, endorsed by, or supported by EnGenius or Senao.\n")
	b.WriteString("Names used only to identify the affected products.\n\n")
	b.WriteString("For " + codeStyle.Render("interoperability and self-hosting on hardware you own") + ".\n")
	b.WriteString(noStyle.Render("Not") + " for warranty fraud, evading paid licensing, or defeating theft protection.\n")
	b.WriteString(dim.Render("No vendor firmware is redistributed here — you supply your own images.\n\n"))
	b.WriteString("License: " + codeStyle.Render("Blue Oak Model License 1.0.0") + "   ·   No warranty; use at your own risk.")
	return b.String()
}

func oneline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 70 {
		s = s[:67] + "…"
	}
	return s
}
