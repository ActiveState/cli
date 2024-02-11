package packages

import (
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type errMsg error

type view struct {
	width     int
	height    int
	content   string
	remaining int
	packages  []string
	ready     bool
	err       error
	viewport  viewport.Model
}

func NewView() (*view, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil, errs.Wrap(err, "Could not get terminal size")
	}

	return &view{
		width:  width,
		height: height,
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
					if v.remaining < 0 {
						v.remaining = 0
					}
					v.remaining--
				}
			}
			return v, nil
		}
	case tea.WindowSizeMsg:
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height-7)
			v.viewport.SetContent(v.content)
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

func (v *view) initialRemaining() {
	currentEntryIndex := 0
	visibleContent := v.viewport.View()
	for i, entry := range v.packages {
		if strings.Contains(visibleContent, entry) && i > currentEntryIndex {
			currentEntryIndex = i + 1
		}
	}

	v.remaining = len(v.packages) - currentEntryIndex - 1
	if v.remaining < 0 {
		v.remaining = 0
	}
}

func (v *view) footerView() string {
	footerText := fmt.Sprintf("... %d more matches, press Down to scroll", v.remaining)
	footerText += fmt.Sprintf("\n%s'%s'", styleBold.Render("For more info run"), styleActionable.Render(" state info <name>"))
	return lipgloss.NewStyle().Render(footerText)
}
