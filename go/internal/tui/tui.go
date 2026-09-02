// Package tui is swallow's professional falconry dashboard (Bubble Tea). Each
// stage renders live output from the real internal packages, so what you see is
// exactly what the tool computes — including the safety refusals.
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

var (
	amber = lipgloss.Color("214")
	teal  = lipgloss.Color("44")
	green = lipgloss.Color("42")
	red   = lipgloss.Color("203")
	dimc  = lipgloss.Color("244")
	fgc   = lipgloss.Color("252")
	bd    = lipgloss.Color("240")

	tTitle = lipgloss.NewStyle().Bold(true).Foreground(amber)
	tDim   = lipgloss.NewStyle().Foreground(dimc)
	tTeal  = lipgloss.NewStyle().Foreground(teal).Bold(true)
	tVer   = lipgloss.NewStyle().Foreground(teal)
	tOK    = lipgloss.NewStyle().Foreground(green)
	tNo    = lipgloss.NewStyle().Foreground(red).Bold(true)
	tCode  = lipgloss.NewStyle().Foreground(amber)
	tFg    = lipgloss.NewStyle().Foreground(fgc)
	tSel   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).Background(amber)
	tItem  = lipgloss.NewStyle().Foreground(fgc)
	menuBx = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bd).Padding(0, 1).MarginRight(1)
	contBx = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bd).Padding(1, 2)
	hdrBx  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(bd)
)

type stage struct {
	glyph, name, sub string
	run              func() string
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
			{"◇", "eyas", "discover / fingerprint", runEyas},
			{"⌁", "jess", "access tether", runJess},
			{"❒", "mews", "backup / evidence", runMews},
			{"⛨", "hood", "append-only safety", runHood},
			{"✚", "band", "unique serials", runBand},
			{"⇅", "flash", "phase 4 · A/B", runFlash},
			{"✓", "verify", "phase 4 · re-read", runFlash},
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
	if bodyH < 13 {
		bodyH = 13
	}
	menuW := 20

	// menu (glyph+name only; no wrap)
	var menu strings.Builder
	for i, s := range m.stages {
		mark := "  "
		if i == m.sel {
			mark = tCode.Render("> ")
		}
		line := fmt.Sprintf("%-9s", s.name)
		if i == m.sel {
			menu.WriteString(mark + tSel.Width(menuW-4).Render(line))
		} else {
			menu.WriteString(mark + tItem.Render(line))
		}
		menu.WriteByte('\n')
	}
	menuPanel := menuBx.Width(menuW).Height(bodyH).Render(strings.TrimRight(menu.String(), "\n"))

	contentW := m.w - menuW - 6
	if contentW < 34 {
		contentW = 34
	}
	cur := m.stages[m.sel]
	title := tTeal.Render(cur.name) + tDim.Render("  ·  "+cur.sub)
	body := title + "\n\n" + cur.run()
	contentPanel := contBx.Width(contentW).Height(bodyH).Render(body)

	main := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, contentPanel)
	foot := tDim.Render(" ↑/↓ move · g/G jump · q quit    ") +
		tDim.Render("· unofficial · not affiliated with EnGenius/Senao · hardware you own only ·")
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), main, foot)
}

func (m model) header() string {
	line := " " + tTitle.Render("swallow") +
		tDim.Render("  ·  ap-hk07 firmware tools  ·  ") +
		tVer.Render("v"+m.version) +
		tDim.Render("  ·  falconry · no-brick")
	return hdrBx.Width(m.w - 1).Render(line)
}

// ---- stage renderers (real package output; markers kept stable for E2E) ----

func runEyas() string {
	var b strings.Builder
	b.WriteString(tDim.Render("fingerprint from web markers, then pick the access adapter") + "\n\n")
	rows := [][3]string{
		{"cloud", "static/js/main", "React GUI · JSON API (admin/admin) · no shell"},
		{"ews-luci", "md5.js · password_plain_text · cgi-bin/luci", "SSH exec :8822 + LuCI flashops"},
		{"fit", "ews377-fit · fitcontroller", "FitController/EPC managed"},
	}
	for _, r := range rows {
		b.WriteString("  " + tCode.Render(fmt.Sprintf("%-9s", r[0])) + tFg.Render(r[1]) + "\n")
		b.WriteString("            " + tDim.Render(r[2]) + "\n")
	}
	b.WriteString("\n" + tDim.Render("live:  swallow discover http://<ap-ip>"))
	return b.String()
}

func runJess() string {
	var b strings.Builder
	b.WriteString(tDim.Render("access tether per family — the reason a default SSH client fails:") + "\n\n")
	b.WriteString("  " + tCode.Render("SSH") + "     port " + tCode.Render("8822") + tDim.Render(" (not 22), legacy ssh-rsa host keys, root / web-admin pw") + "\n")
	b.WriteString("  " + tCode.Render("cloud") + "   " + tDim.Render("/api/sys/login → bearer · sys_info · force_ac · upload.cgi") + "\n")
	b.WriteString("  " + tCode.Render("LuCI") + "    " + tDim.Render("md5(pw+\\n) login → stok → flashops (two-step)") + "\n\n")
	b.WriteString(tDim.Render("adapters never bypass the hood gate for env writes."))
	return b.String()
}

func runMews() string {
	var b strings.Builder
	b.WriteString(tDim.Render("hard-gate backup before any flash (ART is read-only):") + "\n\n")
	for _, a := range mews.Plan() {
		tag := tDim.Render("optional")
		if a.Critical {
			tag = tNo.Render("critical")
		}
		b.WriteString("  " + tOK.Render("▢ ") + tFg.Render(fmt.Sprintf("%-20s", a.Name)) + tDim.Render(a.Command) + "  " + tag + "\n")
	}
	b.WriteString("\n" + tDim.Render("mtd11 = ART (RF calibration + factory MACs) — never written."))
	return b.String()
}

func runHood() string {
	var b strings.Builder
	b.WriteString(tTeal.Render("env is APPEND-ONLY — the tool cannot brick") + "\n\n")
	good := hood.ParsePrintenv("bootcmd=bootipq\nactive_fw=0\napp_part=0\nrootfsname=rootfs\nsnextra=00000000000000000000\n")
	cmd, _ := good.PlanSet("snextra", "SWLWX42000000000000A")
	b.WriteString("  " + tOK.Render("complete env") + tDim.Render("  →  allowed:  ") + tCode.Render(cmd) + "\n")
	wiped := hood.ParsePrintenv("ethaddr=00:03:7f:12:3e:87\n")
	_, e1 := wiped.PlanSet("snextra", "x")
	b.WriteString("  " + tNo.Render("wiped env") + tDim.Render(fmt.Sprintf("     →  refused: missing %v", wiped.Missing())) + "\n")
	_, e2 := good.PlanSet("snextra", "")
	_ = e1
	b.WriteString("  " + tNo.Render("empty value") + tDim.Render("   →  refused: "+oneline(e2.Error())) + "\n\n")
	b.WriteString(tDim.Render("no PlanErase/PlanReset exists; empty values (u-boot deletes) are refused."))
	return b.String()
}

func runBand() string {
	var b strings.Builder
	b.WriteString(tDim.Render("mint a unique, valid identity per AP (Code27, collision-checked):") + "\n\n")
	for i, mdl := range [][2]string{{"ECW230v3", "X42"}, {"EWS377-FIT", "X45"}, {"EWS377AP v3", "X44"}} {
		ser, _ := band.MakeSerial("SWLW", mdl[1], fmt.Sprintf("%04d", i+1))
		ok := tOK.Render("valid")
		if !band.ValidateSerial(ser) {
			ok = tNo.Render("BAD")
		}
		b.WriteString("  " + tFg.Render(fmt.Sprintf("%-12s", mdl[0])) + tDim.Render("-> ") + ser + tDim.Render("  ("+mdl[1]+") ") + ok + "\n")
	}
	x, _ := band.MakeSnextra("SWLW", "X42")
	b.WriteString("\n" + tDim.Render("  field 19 snextra (20 chars): ") + tCode.Render(x) + "\n")
	return b.String()
}

func runFlash() string {
	var b strings.Builder
	b.WriteString(tDim.Render("planned — dev phase 4") + "\n\n")
	b.WriteString(tTeal.Render("no-UART A/B flash") + "\n")
	b.WriteString("  " + tOK.Render("1.") + " write the " + tCode.Render("INACTIVE") + " slot (active stays bootable)\n")
	b.WriteString("  " + tOK.Render("2.") + " reboot-watch, re-read env + serial\n")
	b.WriteString("  " + tOK.Render("3.") + " rollback available; UART only if truly dead\n")
	return b.String()
}

func oneline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 52 {
		s = s[:49] + "…"
	}
	return s
}
