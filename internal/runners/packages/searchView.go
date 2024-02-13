package packages

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	verticalMargin = 7
)

type errMsg error

type view struct {
	width         int
	height        int
	content       string
	remaining     int
	searchResults *structuredSearchResults
	ready         bool
	err           error
	viewport      viewport.Model
}

func NewView(results *structuredSearchResults) (*view, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil, errs.Wrap(err, "Could not get terminal size")
	}

	return &view{
		width:         width,
		height:        height,
		searchResults: results,
	}, nil
}

func (v *view) Init() tea.Cmd {
	return nil
}

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		stringMsg := strings.ToLower(msg.String())
		switch stringMsg {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "up":
			lines := v.viewport.LineUp(1)
			for _, l := range lines {
				if strings.Contains(l, "Name") {
					v.remaining++
				}
			}
			return v, nil
		case "down":
			lines := v.viewport.LineDown(1)
			for _, l := range lines {
				if strings.Contains(l, "Name") {
					if v.remaining <= 0 {
						v.remaining = 0
					} else {
						v.remaining--
					}
				}
			}
			return v, nil
		}
	case tea.WindowSizeMsg:
		if !v.ready {
			// Keep the searching message and command in view
			v.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			v.viewport.SetContent(v.processContent())
			v.initialRemaining()
			v.ready = true
		} else {
			v.width = msg.Width
			v.height = msg.Height
		}
	case errMsg:
		v.err = msg
		return v, nil
	}

	v.viewport, cmd = v.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

func (v *view) View() string {
	if v.err != nil {
		return v.err.Error()
	}
	return v.viewport.View() + "\n\n" + v.footerView()
}

func (v *view) processContent() string {
	maxKeyLength := 0
	for _, key := range keys {
		renderedKey := styleBold.Render(key)
		if len(renderedKey) > maxKeyLength {
			maxKeyLength = len(renderedKey) + 2
		}
	}

	doc := strings.Builder{}
	for _, pkg := range v.searchResults.Results {
		if pkg.Name != "" {
			doc.WriteString(formatRow(styleBold.Render(keyName), pkg.Name, maxKeyLength, v.width))
		}
		if pkg.Description != "" {
			doc.WriteString(formatRow(styleBold.Render(keyDescription), pkg.Description, maxKeyLength, v.width))
		}
		if pkg.Website != "" {
			doc.WriteString(formatRow(styleBold.Render(keyWebsite), styleCyan.Render(pkg.Website), maxKeyLength, v.width))
		}
		if pkg.License != "" {
			doc.WriteString(formatRow(styleBold.Render(keyLicense), pkg.License, maxKeyLength, v.width))
		}

		var versions []string
		for i, v := range pkg.Versions {
			if i > 5 {
				versions = append(versions, locale.Tl("search_more_versions", "... ({{.V0}} more)", strconv.Itoa(len(pkg.Versions)-5)))
				break
			}
			versions = append(versions, styleCyan.Render(v))
		}
		if len(versions) > 0 {
			doc.WriteString(formatRow(styleBold.Render(keyVersions), strings.Join(versions, ", "), maxKeyLength, v.width))
		}

		if len(pkg.Vulnerabilities) > 0 {
			var (
				critical = pkg.Vulnerabilities["Critical"]
				high     = pkg.Vulnerabilities["High"]
				medium   = pkg.Vulnerabilities["Medium"]
				low      = pkg.Vulnerabilities["Low"]
			)

			vunlSummary := []string{}
			if critical > 0 {
				vunlSummary = append(vunlSummary, styleRed.Render(locale.Tl("search_critical", "{{.V0}} Critical", strconv.Itoa(critical))))
			}
			if high > 0 {
				vunlSummary = append(vunlSummary, styleOrange.Render(locale.Tl("search_high", "{{.V0}} High", strconv.Itoa(high))))
			}
			if medium > 0 {
				vunlSummary = append(vunlSummary, styleYellow.Render(locale.Tl("search_medium", "{{.V0}} Medium", strconv.Itoa(medium))))
			}
			if low > 0 {
				vunlSummary = append(vunlSummary, styleMagenta.Render(locale.Tl("search_low", "{{.V0}} Low", strconv.Itoa(low))))
			}

			if len(vunlSummary) > 0 {
				doc.WriteString(formatRow(styleBold.Render(keyVulns), strings.Join(vunlSummary, ", "), maxKeyLength, v.width))
			}
		}

		doc.WriteString("\n")
	}
	return doc.String()
}

func (v *view) initialRemaining() {
	currentEntryIndex := 0
	visibleContent := v.viewport.View()
	for i, entry := range v.searchResults.packageNames {
		if strings.Contains(visibleContent, entry) && i > currentEntryIndex {
			currentEntryIndex = i + 1
		}
	}

	v.remaining = len(v.searchResults.packageNames) - currentEntryIndex - 1
	if v.remaining < 0 {
		v.remaining = 0
	}
}

func (v *view) footerView() string {
	var footerText string
	if v.remaining != 0 {
		footerText += locale.Tl("search_more_matches", "... {{.V0}} more matches, press Down to scroll", strconv.Itoa(v.remaining))
	}
	footerText += fmt.Sprintf("\n%s'%s'", styleBold.Render(locale.Tl("search_more_info", "For more info run")), styleActionable.Render(locale.Tl("search_more_info_command", " state info <name>")))
	return lipgloss.NewStyle().Render(footerText)
}

func formatRow(key, value string, maxKeyLength, width int) string {
	rowStyle := lipgloss.NewStyle().Width(width)

	// Pad key and wrap value
	// The viewport does not support padding so we need to pad the key manually
	paddedKey := strings.Repeat(" ", leftPad) + key + strings.Repeat(" ", maxKeyLength-len(key))
	valueStyle := lipgloss.NewStyle().Width(width - len(paddedKey))

	wrapped := valueStyle.Render(value)

	// The rendered line ends up being a bit too long, so we need to reduce the
	// width that we are working with to ensure that the wrapped value fits
	indentedValue := strings.ReplaceAll(wrapped, "\n", "\n"+strings.Repeat(" ", len(paddedKey)-8))

	formattedRow := fmt.Sprintf("%s%s", paddedKey, indentedValue)
	return rowStyle.Render(formattedRow) + "\n"
}
