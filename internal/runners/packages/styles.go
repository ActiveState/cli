package packages

import "github.com/charmbracelet/lipgloss"

var (
	colorRed        = lipgloss.Color("1")
	colorOrange     = lipgloss.Color("3")
	colorYellow     = lipgloss.Color("4")
	colorMagenta    = lipgloss.Color("5")
	colorCyan       = lipgloss.Color("6")
	styleCyan       = lipgloss.NewStyle().Foreground(colorCyan)
	styleRed        = lipgloss.NewStyle().Foreground(colorRed)
	styleOrange     = lipgloss.NewStyle().Foreground(colorOrange)
	styleYellow     = lipgloss.NewStyle().Foreground(colorYellow)
	styleMagenta    = lipgloss.NewStyle().Foreground(colorMagenta)
	styleBold       = lipgloss.NewStyle().Bold(true)
	styleActionable = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
)
