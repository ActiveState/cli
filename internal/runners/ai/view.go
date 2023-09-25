package ai

import (
	"bytes"
	"strings"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh/terminal"
)

// A view can be more or less any type of data. It holds all the data for a
// program, so often it's a struct. For this simple example, however, all
// we'll need is a simple integer.
type view struct {
	query    string
	spinner  spinner.Model
	packages []*Package
	index    int
	width    int
	height   int
}

func NewView(query string) (*view, error) {
	width, height, err := terminal.GetSize(0)
	if err != nil {
		return nil, err
	}
	spin := spinner.New()
	spin.Style = spinnerStyle
	spin.Spinner = spinner.Meter
	spin.Tick()
	return &view{query, spin, nil, 0, width, height}, nil
}

// Init optionally returns an initial command we should run. In this case we
// want to start the timer.
func (v *view) Init() tea.Cmd {
	return tick
}

// Update is called when messages are received. The idea is that you inspect the
// message and send back an updated model accordingly. You can also return
// a command, which is a function that performs I/O and returns a message.
func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		if m.String() == "ctrl+c" || strings.ToLower(m.String()) == "q" {
			return v, tea.Quit
		}
		if strings.ToLower(m.String()) == "b" && v.index > 0 {
			v.index--
		}
		if strings.ToLower(m.String()) == "n" && v.index < len(v.packages)-1 {
			v.index++
		}
		return v, tick // tea.Quit
	case tickMsg:
		v.spinner, _ = v.spinner.Update(v.spinner.Tick())
		return v, tick
	}
	return v, nil
}

// View returns a string based on data in the model. That string which will be
// rendered to the terminal.
func (v *view) View() string {
	if v.packages == nil {
		return v.viewSpinner()
	}

	pkg := v.packages[v.index]

	doc := strings.Builder{}

	highlighted := bytes.NewBuffer([]byte{})
	quick.Highlight(highlighted, pkg.Example, "python", "terminal256", "native")

	spacing := "   "
	var statusbar string
	if v.index < len(v.packages)-1 {
		msg := "Press N for next result: " + v.packages[v.index+1].Name + "."
		if v.index > 0 {
			msg += spacing + "Press B for previous result: " + v.packages[v.index-1].Name + "."
		}
		msg += spacing + "Press Q to quit."
		statusbar = styleStatusBar.Width(v.width).Render(msg)
	} else {
		statusbar = styleStatusBar.Width(v.width).Render(" End reached." + spacing + "Press Q to quit.")
	}

	doc.WriteString(
		lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.JoinVertical(lipgloss.Top,
					lipgloss.Place(v.width/2, 0, lipgloss.Left, lipgloss.Top,
						combine(
							styleNotice.Inherit(stylePad).Render("Name:"),
							stylePad.Padding(1).Render(pkg.Name),

							styleNotice.Inherit(stylePad).Render("Description:"),
							stylePad.Padding(1).Width(v.width/2-10).Render(pkg.Description),

							styleNotice.Inherit(stylePad).Render("Example:"),
							stylePad.Padding(1).Render(highlighted.String()),
						))),
				lipgloss.NewStyle().MarginLeft(2).Render(
					lipgloss.JoinVertical(lipgloss.Top,
						lipgloss.Place(v.width/2-20, 0, lipgloss.Left, lipgloss.Top,
							combine(
								styleNotice.Inherit(stylePad).Render("Pros:"),
								stylePad.Padding(1).Render(listBullet+strings.Join(pkg.Advantages, "\n"+listBullet)),

								styleNotice.Inherit(stylePad).Render("Cons:"),
								stylePad.Padding(1).Render(listBullet+strings.Join(pkg.Disadvantages, "\n"+listBullet)),

								styleNotice.Inherit(stylePad).Render("Used in:"),
								stylePad.Padding(1).Render(listBullet+strings.Join(pkg.Projects, "\n"+listBullet)),
							),
						))),
			),
			statusbar,
		))

	return styleDoc.Render(doc.String())
}

func (v *view) viewSpinner() string {
	dialog := lipgloss.Place(v.width, 2,
		lipgloss.Center, lipgloss.Center,
		styleDialog.Render(lipgloss.JoinVertical(lipgloss.Center,
			styleNotice.Render("Searching for Python packages matching the query:"),
			"",
			styleAction.Render(v.query),
			"",
			v.spinner.View(),
		)),
	)
	return dialog
}

// Messages are events that we respond to in our Update function. This
// particular one indicates that the timer has ticked.
type tickMsg time.Time

func tick() tea.Msg {
	time.Sleep(time.Millisecond * 200)
	return tickMsg{}
}

func combine(v ...string) string {
	return strings.Join(v, "\n")
}
