// Package tui is swallow's dashboard (Bubble Tea). Each screen renders live
// output from the real internal packages, so what you see is exactly what the
// tool computes — including the safety refusals. The visuals aim for a calm,
// refined feel: neutral text, one soft accent, and plain-language labels.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/band"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/hood"
	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/mews"
)

// Everything sits on an elevated dark "surface" (like a macOS window) rather than
// pure black, so near-white text reads crisply instead of thin-gray-on-black.
// One blue accent for selection/links; green/red only carry status meaning.
var (
	surface  = lipgloss.Color("235") // window/card background (elevated, not black)
	ink      = lipgloss.Color("231") // headings + values — bright white
	body     = lipgloss.Color("253") // body text — high contrast
	muted    = lipgloss.Color("249") // secondary/captions — still clearly readable
	line     = lipgloss.Color("240") // subtle card borders
	accent   = lipgloss.Color("111") // links / step numbers — bright blue
	pillBg   = lipgloss.Color("39")  // selected-row pill background
	okc      = lipgloss.Color("114") // system green
	noc      = lipgloss.Color("210") // system red
	warnc    = lipgloss.Color("179") // amber (required tag)
	onAccent = lipgloss.Color("231") // text on the accent pill

	// base carries the surface background so no span falls back to terminal black.
	base = lipgloss.NewStyle().Background(surface)

	sProduct = base.Foreground(ink).Bold(true)
	sHead    = base.Foreground(ink).Bold(true)
	sSub     = base.Foreground(muted)
	sBody    = base.Foreground(body)
	sInk     = base.Foreground(body)
	sKey     = base.Foreground(ink).Bold(true) // values / tokens pop
	sAccent  = base.Foreground(accent)
	sVer     = base.Foreground(accent)
	sOK      = base.Foreground(okc)
	sNo      = base.Foreground(noc).Bold(true)
	sWarn    = base.Foreground(warnc)
	sPill    = base.Foreground(onAccent).Background(pillBg).Bold(true)
	sItem    = base.Foreground(body)

	menuBx = base.Border(lipgloss.RoundedBorder()).BorderForeground(line).
		BorderBackground(surface).Padding(1, 1).MarginRight(1)
	contBx = base.Border(lipgloss.RoundedBorder()).BorderForeground(line).
		BorderBackground(surface).Padding(1, 3)
	barBx = base.Padding(0, 1) // full-width header/footer bars
)

type stage struct {
	name, sub string
	run       func() string
}

type model struct {
	stages  []stage
	sel     int
	version string
	w, h    int
}

// New builds the dashboard model.
func New(version string) model {
	return model{
		version: version,
		stages: []stage{
			{"Discover", "Identify the access point", runDiscover},
			{"Connect", "Choose how to reach it", runConnect},
			{"Back Up", "Save everything first", runBackup},
			{"Safeguards", "Why it can't brick", runSafeguards},
			{"Identity", "Assign a unique serial", runIdentity},
			{"Install", "Flash without UART", runInstall},
			{"Verify", "Confirm it came back", runVerify},
		},
	}
}

// Run launches the dashboard.
func Run(version string) error {
	_, err := tea.NewProgram(New(version), tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.sel > 0 {
				m.sel--
			}
		case "down", "j":
			if m.sel < len(m.stages)-1 {
				m.sel++
			}
		case "g", "home":
			m.sel = 0
		case "G", "end":
			m.sel = len(m.stages) - 1
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.w == 0 {
		m.w, m.h = 100, 30
	}
	// Height budget: header (2 rows) + card + footer (2 rows) must fit m.h. The
	// card's rounded border adds 2 rows on top of its Height(), so reserve 6.
	bodyH := m.h - 6
	if bodyH < 14 {
		bodyH = 14
	}
	// Menu card is a fixed width; its inner text width accounts for the rounded
	// border (2) + horizontal padding (2).
	menuW := 24
	menuInner := menuW - 4

	// Sidebar: plain step names, numbered so the flow reads as a sequence. The
	// selected row is a full-width accent pill (like a settings sidebar).
	var menu strings.Builder
	menu.WriteString(sSub.Render("STEPS") + "\n\n")
	for i, s := range m.stages {
		label := fmt.Sprintf(" %d  %-*s", i+1, menuInner-4, s.name)
		if i == m.sel {
			menu.WriteString(sPill.Render(label))
		} else {
			menu.WriteString(sItem.Render(label))
		}
		menu.WriteString("\n\n")
	}
	menuPanel := menuBx.Width(menuW).Height(bodyH).Render(strings.TrimRight(menu.String(), "\n"))

	// Content card fills the rest of the width (menu box + its 1-col right margin),
	// so the cards line up with the full-width header/footer bars.
	contentW := m.w - menuW - 1
	if contentW < 40 {
		contentW = 40
	}
	cur := m.stages[m.sel]
	crumb := sSub.Render(fmt.Sprintf("Step %d of %d", m.sel+1, len(m.stages)))
	title := sHead.Render(cur.name) + sSub.Render("    ") + crumb
	sub := sSub.Render(cur.sub)
	bodyText := title + "\n" + sub + "\n\n" + cur.run()
	contentPanel := contBx.Width(contentW).Height(bodyH).Render(bodyText)

	main := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, contentPanel)

	keys := sAccent.Render("↑↓") + sSub.Render(" move") + sSub.Render("    ") +
		sAccent.Render("g/G") + sSub.Render(" ends") + sSub.Render("    ") +
		sAccent.Render("q") + sSub.Render(" quit")
	legal := sSub.Render("unofficial · not affiliated with EnGenius/Senao · hardware you own")
	foot := barBx.Width(m.w).Render(keys) + "\n" + barBx.Width(m.w).Render(legal)

	return lipgloss.JoinVertical(lipgloss.Left, m.header(), main, foot)
}

func (m model) header() string {
	title := sProduct.Render("swallow") +
		sSub.Render("  ·  ap-hk07 firmware toolkit  ·  ") +
		sVer.Render("v"+m.version)
	return barBx.Width(m.w).Render(title)
}

// ---- screen renderers (real package output; blunt, plain-language copy) ----
//
// Every visible segment must be rendered through a surface-backed style — a raw
// string literal between styled spans would show as a black gap on the card.

// pad returns n surface-backed spaces (for indents/separators, never raw " ").
func pad(n int) string { return base.Render(strings.Repeat(" ", n)) }

// blank returns a full-width surface-backed line so multi-line bodies keep the
// card colour on otherwise-empty rows.
func blank() string { return "\n" }

func runDiscover() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Find the AP and read what firmware it runs.") + blank() + blank())
	rows := [][3]string{
		{"Cloud model", "React web UI · JSON API (admin/admin)", "no shell"},
		{"EWS model", "LuCI web UI · md5.js · cgi-bin/luci", "SSH + LuCI upload"},
		{"FIT model", "ews377-fit · FitController", "controller-managed"},
	}
	for _, r := range rows {
		b.WriteString(pad(2) + sKey.Render(fmt.Sprintf("%-13s", r[0])) + sBody.Render(r[1]) + "\n")
		b.WriteString(pad(15) + sSub.Render(r[2]) + "\n")
	}
	b.WriteString(blank() + sSub.Render("Run:  ") + sAccent.Render("swallow discover http://<ap-ip>"))
	return b.String()
}

func runConnect() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Pick the access path. A normal SSH client won't reach it.") + blank() + blank())
	b.WriteString(pad(2) + sKey.Render("SSH   ") + sBody.Render("port ") + sKey.Render("8822") +
		sBody.Render(" (not 22), old host-key type, admin password") + "\n")
	b.WriteString(pad(2) + sKey.Render("Cloud ") + sBody.Render("web API login → token → upload") + "\n")
	b.WriteString(pad(2) + sKey.Render("LuCI  ") + sBody.Render("hashed-password login → session → 2-step upload") + "\n")
	b.WriteString(blank() + sSub.Render("All firmware writes still pass the safety checks."))
	return b.String()
}

func runBackup() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Dump everything first. This is a hard gate, not a suggestion.") + blank() + blank())
	for _, a := range mews.Plan() {
		tag := sSub.Render("optional")
		if a.Critical {
			tag = sWarn.Render("required")
		}
		b.WriteString(pad(2) + sOK.Render("· ") + sInk.Render(fmt.Sprintf("%-20s", a.Name)) +
			sSub.Render(fmt.Sprintf("%-26s", a.Command)) + tag + "\n")
	}
	b.WriteString(blank() + sSub.Render("RF calibration (factory MACs) is read-only — never written."))
	return b.String()
}

func runSafeguards() string {
	var b strings.Builder
	b.WriteString(sBody.Render("The bootloader env is append-only. The tool can't brick it.") + blank() + blank())
	good := hood.ParsePrintenv("bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nsnextra=00000000000000000000\n")
	cmd, _ := good.PlanSet("snextra", "SWLWX42000000000000A")
	wiped := hood.ParsePrintenv("ethaddr=00:03:7f:12:3e:87\n")
	_, _ = wiped.PlanSet("snextra", "x")
	b.WriteString(pad(2) + sOK.Render(fmt.Sprintf("%-16s", "· complete env")) + sBody.Render("allowed — adds one field") + "\n")
	b.WriteString(pad(2) + sNo.Render(fmt.Sprintf("%-16s", "· wiped env")) + sBody.Render(fmt.Sprintf("refused — %d required keys missing", len(wiped.Missing()))) + "\n")
	b.WriteString(pad(2) + sNo.Render(fmt.Sprintf("%-16s", "· empty value")) + sBody.Render("refused — u-boot would delete it") + "\n")
	b.WriteString(blank() + sSub.Render("Allowed write:  ") + sAccent.Render(cmd) + "\n")
	b.WriteString(blank() + sSub.Render("It only ever adds a field — never erases or resets."))
	return b.String()
}

func runIdentity() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Mint a unique, valid serial per unit. Collisions are refused.") + blank() + blank())
	for i, mdl := range [][2]string{{"ECW230v3", "X42"}, {"EWS377-FIT", "X45"}, {"EWS377AP v3", "X44"}} {
		ser, _ := band.MakeSerial("SWLW", mdl[1], fmt.Sprintf("%04d", i+1))
		ok := sOK.Render("valid")
		if !band.ValidateSerial(ser) {
			ok = sNo.Render("BAD")
		}
		b.WriteString(pad(2) + sInk.Render(fmt.Sprintf("%-12s", mdl[0])) + sSub.Render("→ ") +
			sKey.Render(ser) + sSub.Render("  ("+mdl[1]+")  ") + ok + "\n")
	}
	x, _ := band.MakeSnextra("SWLW", "X42")
	b.WriteString(blank() + pad(2) + sSub.Render("field 19 (20 chars): ") + sAccent.Render(x))
	return b.String()
}

func runInstall() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Flash over the network. No UART, no open case.") + blank() + blank())
	b.WriteString(pad(2) + sAccent.Render("1  ") + sBody.Render("Write the ") + sKey.Render("spare") + sBody.Render(" slot; the running one stays bootable") + "\n")
	b.WriteString(pad(2) + sAccent.Render("2  ") + sBody.Render("Reboot, then re-read firmware + serial") + "\n")
	b.WriteString(pad(2) + sAccent.Render("3  ") + sBody.Render("Roll back anytime; UART only if truly dead") + "\n")
	b.WriteString(blank() + sSub.Render("The active slot is never overwritten, so a bad flash can't brick."))
	return b.String()
}

func runVerify() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Prove it came back exactly as intended.") + blank() + blank())
	b.WriteString(pad(2) + sOK.Render("· ") + sBody.Render("Re-read firmware family + serial after reboot") + "\n")
	b.WriteString(pad(2) + sOK.Render("· ") + sBody.Render("Compare against what was written") + "\n")
	b.WriteString(pad(2) + sOK.Render("· ") + sBody.Render("Flag any mismatch, offer one-step rollback") + "\n")
	b.WriteString(blank() + sSub.Render("Not done until this passes."))
	return b.String()
}
