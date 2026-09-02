package tui

import "github.com/charmbracelet/lipgloss"

// Falconry palette — amber raptor on slate, teal accents.
var (
	amber = lipgloss.Color("214")
	teal  = lipgloss.Color("44")
	slate = lipgloss.Color("240")
	fg    = lipgloss.Color("252")
	muted = lipgloss.Color("245")
	green = lipgloss.Color("42")
	red   = lipgloss.Color("203")
	yellow= lipgloss.Color("220")
	bg    = lipgloss.Color("236")

	titleStyle   = lipgloss.NewStyle().Foreground(amber).Bold(true)
	tagStyle     = lipgloss.NewStyle().Foreground(muted).Italic(true)
	verStyle     = lipgloss.NewStyle().Foreground(teal)
	badgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Background(amber).Bold(true).Padding(0, 1)
	menuBox      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(slate).Padding(0, 1)
	contentBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(slate).Padding(1, 2)
	headerBox    = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(slate).Padding(0, 1)
	itemStyle    = lipgloss.NewStyle().Foreground(fg).Padding(0, 1)
	itemSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Background(amber).Bold(true).Padding(0, 1)
	h2           = lipgloss.NewStyle().Foreground(teal).Bold(true)
	dim          = lipgloss.NewStyle().Foreground(muted)
	okStyle      = lipgloss.NewStyle().Foreground(green)
	noStyle      = lipgloss.NewStyle().Foreground(red).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(yellow)
	codeStyle    = lipgloss.NewStyle().Foreground(amber)
	footStyle    = lipgloss.NewStyle().Foreground(muted).Padding(0, 1)
)
