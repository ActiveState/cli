package packages

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	keyName        = locale.Tl("search_name", "Name")
	keyDescription = locale.Tl("search_description", "Description")
	keyWebsite     = locale.Tl("search_website", "Website")
	keyLicense     = locale.Tl("search_license", "License")
	keyVersions    = locale.Tl("search_versions", "Versions")
	keyVulns       = locale.Tl("search_vulnerabilities", "Vulnerabilities (CVEs)")

	keys = []string{
		keyName,
		keyDescription,
		keyWebsite,
		keyLicense,
		keyVersions,
		keyVulns,
	}
)

const (
	leftPad        = 2
	verticalMargin = 7
	scrollUp       = "up"
	scrollDown     = "down"
)

type view struct {
	width         int
	height        int
	content       string
	searchResults *structuredSearchResults
	ready         bool
	viewport      viewport.Model
}

func NewView(results *structuredSearchResults, out output.Outputer) (*view, error) {
	outFD, ok := out.Config().OutWriterFD()
	if !ok {
		logging.Error("Could not get output writer file descriptor, falling back to stdout")
		outFD = os.Stdout.Fd()
	}

	width, height, err := term.GetSize(int(outFD))
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
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.viewport.LineUp(3)
			return v, nil
		case tea.MouseButtonWheelDown:
			v.viewport.LineDown(3)
			return v, nil
		}
	case tea.KeyMsg:
		stringMsg := strings.ToLower(msg.String())
		switch stringMsg {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "up":
			v.viewport.LineUp(1)
			return v, nil
		case "down":
			v.viewport.LineDown(1)
			return v, nil
		case "pgup":
			v.viewport.LineUp(v.height - verticalMargin)
			return v, nil
		case "pgdown":
			v.viewport.LineDown(v.height - verticalMargin)
			return v, nil
		}
	case tea.WindowSizeMsg:
		if !v.ready {
			// Keep the searching message and command in view
			v.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			v.content = v.processContent()
			v.viewport.SetContent(v.processContent())
			v.ready = true
		} else {
			v.width = msg.Width
			v.height = msg.Height
		}
	}

	v.viewport, cmd = v.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

func (v *view) View() string {
	return v.viewport.View() + "\n\n" + v.footerView()
}

func (v *view) processContent() string {
	maxKeyLength := 0
	for _, key := range keys {
		renderedKey := colorize.StyleLightGrey.Render(key)
		if len(renderedKey) > maxKeyLength {
			maxKeyLength = len(renderedKey) + 2
		}
	}

	doc := strings.Builder{}
	for _, pkg := range v.searchResults.Results {
		if pkg.Name != "" {
			doc.WriteString(formatRow(colorize.StyleLightGrey.Render(keyName), colorize.StyleActionable.Render(pkg.Name), maxKeyLength, v.width))
		}
		if pkg.Description != "" {
			doc.WriteString(formatRow(colorize.StyleLightGrey.Render(keyDescription), pkg.Description, maxKeyLength, v.width))
		}
		if pkg.Website != "" {
			doc.WriteString(formatRow(colorize.StyleLightGrey.Render(keyWebsite), colorize.StyleCyan.Render(pkg.Website), maxKeyLength, v.width))
		}
		if pkg.License != "" {
			doc.WriteString(formatRow(colorize.StyleLightGrey.Render(keyLicense), colorize.StyleCyan.Render(pkg.License), maxKeyLength, v.width))
		}

		var versions []string
		for i, v := range pkg.Versions {
			if i > 5 {
				versions = append(versions, locale.Tl("search_more_versions", "... ({{.V0}} more)", strconv.Itoa(len(pkg.Versions)-5)))
				break
			}
			versions = append(versions, colorize.StyleCyan.Render(v))
		}
		if len(versions) > 0 {
			doc.WriteString(formatRow(colorize.StyleLightGrey.Render(keyVersions), strings.Join(versions, ", "), maxKeyLength, v.width))
		}

		if len(pkg.Vulnerabilities) > 0 {
			var (
				critical = pkg.Vulnerabilities[model.SeverityCritical]
				high     = pkg.Vulnerabilities[model.SeverityHigh]
				medium   = pkg.Vulnerabilities[model.SeverityMedium]
				low      = pkg.Vulnerabilities[model.SeverityLow]
			)

			vunlSummary := []string{}
			if critical > 0 {
				vunlSummary = append(vunlSummary, colorize.StyleRed.Render(locale.Tl("search_critical", "{{.V0}} Critical", strconv.Itoa(critical))))
			}
			if high > 0 {
				vunlSummary = append(vunlSummary, colorize.StyleOrange.Render(locale.Tl("search_high", "{{.V0}} High", strconv.Itoa(high))))
			}
			if medium > 0 {
				vunlSummary = append(vunlSummary, colorize.StyleYellow.Render(locale.Tl("search_medium", "{{.V0}} Medium", strconv.Itoa(medium))))
			}
			if low > 0 {
				vunlSummary = append(vunlSummary, colorize.StyleMagenta.Render(locale.Tl("search_low", "{{.V0}} Low", strconv.Itoa(low))))
			}

			if len(vunlSummary) > 0 {
				doc.WriteString(formatRow(colorize.StyleBold.Render(keyVulns), strings.Join(vunlSummary, ", "), maxKeyLength, v.width))
			}
		}

		doc.WriteString("\n")
	}
	return doc.String()
}

func (v *view) footerView() string {
	var footerText string
	scrollValue := v.viewport.ScrollPercent() * 100
	footerText += locale.Tl("search_more_matches", "... {{.V0}}% scrolled, use arrow and page keys to scroll. Press Q to quit.", strconv.Itoa(int(scrollValue)))
	footerText += fmt.Sprintf("\n\n%s '%s'", colorize.StyleBold.Render(locale.Tl("search_more_info", "For more info run")), colorize.StyleActionable.Render(locale.Tl("search_more_info_command", "state info <name>")))
	return lipgloss.NewStyle().Render(footerText)
}

// formatRow formats a key-value pair into a single line of text
// It pads the key both left and right and ensures the value is wrapped to the
// correct width.
// Example:
// Initially we would have:
//
// Name: value
//
// After padding:
//
//	Name:   value
func formatRow(key, value string, maxKeyLength, width int) string {
	rowStyle := lipgloss.NewStyle().Width(width)

	// Pad key and wrap the value
	// The viewport does not support padding so we need to pad the key manually
	// First, pad the key left to indent the entire view
	// Then, pad the key right to ensure that the values are aligned with the
	// other values in the view.
	paddedKey := strings.Repeat(" ", leftPad) + key + strings.Repeat(" ", maxKeyLength-len(key))

	// The value style is strictly for the information that a key maps to.
	// ie. the description string, the website string, etc.
	// We have a separate width here to ensure that the value is wrapped to the
	// correct width.
	valueStyle := lipgloss.NewStyle().Width(width - len(paddedKey))
	wrapped := valueStyle.Render(value)

	// The rendered value ends up being a bit too wide, so we need to reduce the
	// width that we are working with to ensure that the wrapped value fits
	indentedValue := strings.ReplaceAll(wrapped, "\n", "\n"+strings.Repeat(" ", len(paddedKey)-15))

	formattedRow := fmt.Sprintf("%s%s", paddedKey, indentedValue)
	return rowStyle.Render(formattedRow) + "\n"
}
