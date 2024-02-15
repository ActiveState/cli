package colorize

import "github.com/charmbracelet/lipgloss"

var (
	ColorRed        = lipgloss.Color("1")
	ColorOrange     = lipgloss.Color("3")
	ColorYellow     = lipgloss.Color("4")
	ColorMagenta    = lipgloss.Color("5")
	ColorCyan       = lipgloss.Color("6")
	ColorLightGrey  = lipgloss.Color("248")
	StyleCyan       = lipgloss.NewStyle().Foreground(ColorCyan)
	StyleRed        = lipgloss.NewStyle().Foreground(ColorRed)
	StyleOrange     = lipgloss.NewStyle().Foreground(ColorOrange)
	StyleYellow     = lipgloss.NewStyle().Foreground(ColorYellow)
	StyleMagenta    = lipgloss.NewStyle().Foreground(ColorMagenta)
	StyleLightGrey  = lipgloss.NewStyle().Foreground(ColorLightGrey)
	StyleBold       = lipgloss.NewStyle().Bold(true)
	StyleActionable = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
)
