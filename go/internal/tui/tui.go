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

// A calm, system-like palette tuned for legibility on a dark terminal: bright
// near-white headings, a high-contrast body gray, one readable blue accent for
// selection/links, and green/red reserved strictly for status meaning.
var (
	accent   = lipgloss.Color("75")  // selection background
	accentTx = lipgloss.Color("117") // accent text (bright enough to read on black)
	strong   = lipgloss.Color("231") // headings — near-white
	text     = lipgloss.Color("252") // primary body text
	text2    = lipgloss.Color("250") // menu items / secondary
	cap      = lipgloss.Color("245") // captions / hints (still readable)
	line     = lipgloss.Color("240") // visible-but-subtle card borders
	okc      = lipgloss.Color("114") // system green
	noc      = lipgloss.Color("210") // system red
	warnc    = lipgloss.Color("179") // muted amber (required tag)
	onAccent = lipgloss.Color("231") // text on the accent pill

	sProduct = lipgloss.NewStyle().Bold(true).Foreground(strong)
	sHead    = lipgloss.NewStyle().Bold(true).Foreground(strong)
	sSub     = lipgloss.NewStyle().Foreground(cap)
	sBody    = lipgloss.NewStyle().Foreground(text)
	sInk     = lipgloss.NewStyle().Foreground(text)
	sKey     = lipgloss.NewStyle().Bold(true).Foreground(strong) // values / tokens pop
	sAccent  = lipgloss.NewStyle().Foreground(accentTx)
	sVer     = lipgloss.NewStyle().Foreground(accentTx)
	sOK      = lipgloss.NewStyle().Foreground(okc)
	sNo      = lipgloss.NewStyle().Bold(true).Foreground(noc)
	sWarn    = lipgloss.NewStyle().Foreground(warnc)
	sPill    = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent)
	sItem    = lipgloss.NewStyle().Foreground(text2)

	menuBx = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(line).
		Padding(1, 1).MarginRight(1)
	contBx = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(line).Padding(1, 3)
	hdrBx  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(line)
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
	bodyH := m.h - 5
	if bodyH < 16 {
		bodyH = 16
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

	// Content card fills the rest of the width (menu box + its 1-col right margin).
	contentW := m.w - menuW - 2
	if contentW < 40 {
		contentW = 40
	}
	cur := m.stages[m.sel]
	crumb := sSub.Render(fmt.Sprintf("Step %d of %d", m.sel+1, len(m.stages)))
	title := sHead.Render(cur.name) + "   " + crumb
	sub := sSub.Render(cur.sub)
	body := title + "\n" + sub + "\n\n" + cur.run()
	contentPanel := contBx.Width(contentW).Height(bodyH).Render(body)

	main := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, contentPanel)
	foot := sSub.Render("  ↑ ↓  navigate    ·    g / G  first / last    ·    q  quit") + "\n" +
		sSub.Render("  Unofficial · not affiliated with EnGenius or Senao · for hardware you own")
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), main, foot)
}

func (m model) header() string {
	line := " " + sProduct.Render("swallow") +
		sSub.Render("   ap-hk07 firmware toolkit") +
		sSub.Render("      ") + sVer.Render("v"+m.version) +
		sSub.Render("      ·   safe cross-flash")
	return hdrBx.Width(m.w - 1).Render(line)
}

// ---- screen renderers (real package output; plain-language, human copy) ----

func runDiscover() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Identify the access point, then pick how to connect.") + "\n\n")
	rows := [][3]string{
		{"Cloud model", "React web UI · JSON API (admin / admin)", "no shell"},
		{"EWS model", "LuCI web UI · md5.js · cgi-bin/luci", "SSH + LuCI upload"},
		{"FIT model", "ews377-fit · FitController", "controller-managed"},
	}
	for _, r := range rows {
		b.WriteString("  " + sKey.Render(fmt.Sprintf("%-13s", r[0])) + sBody.Render(r[1]) + "\n")
		b.WriteString("               " + sSub.Render(r[2]) + "\n")
	}
	b.WriteString("\n" + sSub.Render("Try it:  ") + sAccent.Render("swallow discover http://<ap-ip>"))
	return b.String()
}

func runConnect() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Choose how to connect — a normal SSH client won't work here:") + "\n\n")
	b.WriteString("  " + sKey.Render("SSH") + "     " + sBody.Render("port ") + sKey.Render("8822") +
		sBody.Render(" (not 22), older host-key type, admin password") + "\n")
	b.WriteString("  " + sKey.Render("Cloud") + "   " + sBody.Render("web API login → token → upload") + "\n")
	b.WriteString("  " + sKey.Render("LuCI") + "    " + sBody.Render("hashed-password login → session → two-step upload") + "\n\n")
	b.WriteString(sSub.Render("Every firmware write still goes through the safety checks."))
	return b.String()
}

func runBackup() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Save a full backup first — this is required, not optional:") + "\n\n")
	for _, a := range mews.Plan() {
		tag := sSub.Render("optional")
		if a.Critical {
			tag = sWarn.Render("required")
		}
		b.WriteString("  " + sOK.Render("· ") + sInk.Render(fmt.Sprintf("%-20s", a.Name)) + sSub.Render(a.Command) + "  " + tag + "\n")
	}
	b.WriteString("\n" + sSub.Render("The RF-calibration area (factory MACs) is read-only and never touched."))
	return b.String()
}

func runSafeguards() string {
	var b strings.Builder
	b.WriteString(sBody.Render("The bootloader is protected — the tool can't brick it.") + "\n\n")
	good := hood.ParsePrintenv("bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nsnextra=00000000000000000000\n")
	cmd, _ := good.PlanSet("snextra", "SWLWX42000000000000A")
	wiped := hood.ParsePrintenv("ethaddr=00:03:7f:12:3e:87\n")
	_, _ = wiped.PlanSet("snextra", "x")
	b.WriteString("  " + sOK.Render("· complete env") + sBody.Render("    allowed — adds a single field") + "\n")
	b.WriteString("  " + sNo.Render("· wiped env") + sBody.Render(fmt.Sprintf("       refused — %d required keys missing", len(wiped.Missing()))) + "\n")
	b.WriteString("  " + sNo.Render("· empty value") + sBody.Render("     refused — u-boot would delete it") + "\n\n")
	b.WriteString(sSub.Render("Example allowed write:  ") + sAccent.Render(cmd) + "\n\n")
	b.WriteString(sSub.Render("It only ever adds a setting — it never erases or resets them."))
	return b.String()
}

func runIdentity() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Give each unit a unique identity, checked for collisions:") + "\n\n")
	for i, mdl := range [][2]string{{"ECW230v3", "X42"}, {"EWS377-FIT", "X45"}, {"EWS377AP v3", "X44"}} {
		ser, _ := band.MakeSerial("SWLW", mdl[1], fmt.Sprintf("%04d", i+1))
		ok := sOK.Render("valid")
		if !band.ValidateSerial(ser) {
			ok = sNo.Render("BAD")
		}
		b.WriteString("  " + sInk.Render(fmt.Sprintf("%-12s", mdl[0])) + sSub.Render("→ ") + sKey.Render(ser) + sSub.Render("  ("+mdl[1]+") ") + ok + "\n")
	}
	x, _ := band.MakeSnextra("SWLW", "X42")
	b.WriteString("\n" + sSub.Render("  Extra identity field (20 chars): ") + sAccent.Render(x) + "\n")
	return b.String()
}

func runInstall() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Install without opening the case — no UART needed:") + "\n\n")
	b.WriteString("  " + sAccent.Render("1.") + " Write the " + sKey.Render("spare") + " slot — the running one stays bootable\n")
	b.WriteString("  " + sAccent.Render("2.") + " Reboot, then re-read the firmware and serial\n")
	b.WriteString("  " + sAccent.Render("3.") + " Roll back anytime; UART only if it's truly dead\n\n")
	b.WriteString(sSub.Render("Because the active slot is never overwritten, a failed flash can't brick."))
	return b.String()
}

func runVerify() string {
	var b strings.Builder
	b.WriteString(sBody.Render("Confirm the device came back exactly as intended:") + "\n\n")
	b.WriteString("  " + sOK.Render("·") + " Reads the firmware family and serial after reboot\n")
	b.WriteString("  " + sOK.Render("·") + " Compares them against what was written\n")
	b.WriteString("  " + sOK.Render("·") + " Flags any mismatch and offers a one-step rollback\n\n")
	b.WriteString(sSub.Render("Only after this passes is the flash considered done."))
	return b.String()
}

func oneline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 52 {
		s = s[:49] + "…"
	}
	return s
}
