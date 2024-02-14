package packages

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
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

type errMsg error

type view struct {
	width         int
	height        int
	remaining     int
	content       string
	index         map[string]int
	searchResults *structuredSearchResults
	ready         bool
	err           error
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
			v.scroll(scrollUp, 3)
			return v, nil
		case tea.MouseButtonWheelDown:
			v.scroll(scrollDown, 3)
			return v, nil
		}
	case tea.KeyMsg:
		stringMsg := strings.ToLower(msg.String())
		switch stringMsg {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "up":
			v.scroll(stringMsg, 1)
			return v, nil
		case "down":
			v.scroll(stringMsg, 1)
			return v, nil
		case "pgup":
			v.scroll("up", v.height-verticalMargin)
			return v, nil
		case "pgdown":
			v.scroll("down", v.height-verticalMargin)
			return v, nil
		}
	case tea.WindowSizeMsg:
		if !v.ready {
			// Keep the searching message and command in view
			v.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			v.content = v.processContent()
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
	// Interate over all of the content and calculate the index.
	// Each time we encounter a new package name, we determine how many
	// remaining packages there are and store that in the index.
	lines := strings.Split(v.content, "\n")
	index := 1
	v.index = make(map[string]int)
	for _, l := range lines {
		if strings.Contains(l, "Name") {
			v.index[l] = index
			index++
		}
	}

	visibleContent := strings.Split(v.viewport.View(), "\n")
	for _, l := range visibleContent {
		if strings.Contains(l, "Name") {
			v.remaining = len(v.searchResults.Results) - v.index[l]
		}
	}

	if v.remaining < 0 {
		v.remaining = 0
	}
}

func (v *view) footerView() string {
	var footerText string
	if v.remaining != 0 {
		footerText += locale.Tl("search_more_matches", "... {{.V0}} more matches, use arrow and page keys to scroll. Press Q to quit.", strconv.Itoa(v.remaining))
	}
	footerText += fmt.Sprintf("\n\n%s'%s'", styleBold.Render(locale.Tl("search_more_info", "For more info run")), styleActionable.Render(locale.Tl("search_more_info_command", " state info <name>")))
	return lipgloss.NewStyle().Render(footerText)
}

func (v *view) scroll(direction string, amount int) {
	if direction == "up" {
		// LineUp returns the new lines. In order to update the remaining count
		// we need to iterate over the visible lines and find the remaining count
		// for the last visible package name.
		_ = v.viewport.LineUp(amount)
		lines := strings.Split(v.viewport.View(), "\n")
		for _, l := range lines {
			if strings.Contains(l, "Name") {
				v.remaining = len(v.searchResults.packageNames) - v.index[l]
			}
		}
	} else {
		lines := v.viewport.LineDown(amount)
		for _, l := range lines {
			if strings.Contains(l, "Name") {
				v.remaining = len(v.searchResults.packageNames) - v.index[l]
			}
		}
	}
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
	indentedValue := strings.ReplaceAll(wrapped, "\n", "\n"+strings.Repeat(" ", len(paddedKey)-8))

	formattedRow := fmt.Sprintf("%s%s", paddedKey, indentedValue)
	return rowStyle.Render(formattedRow) + "\n"
}
