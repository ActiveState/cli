package packages

import "github.com/charmbracelet/lipgloss"

var colorCTA = lipgloss.Color("#8860FF")
var colorAction = lipgloss.Color("#FF457F")

var styleDoc = lipgloss.NewStyle().Padding(1, 2, 1, 2)

var styleNotice = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorCTA)

var styleAction = lipgloss.NewStyle().
	Bold(true).
	Underline(true).
	Foreground(colorAction)

var styleStatusBar = lipgloss.NewStyle().
	Background(colorCTA).
	Foreground(lipgloss.Color("#F0F0F0"))

var listBullet = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorAction).Render(" • ")

var stylePad = lipgloss.NewStyle().Padding(1)

var styleDialog = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorCTA).
	Padding(2, 10).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true)

var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCTA))
