package packages

import (
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh/terminal"
)

type errMsg error

type view struct {
	width    int
	height   int
	content  string
	ready    bool
	viewport viewport.Model
	spinner  spinner.Model
	err      error
}

func NewView() (*view, error) {
	width, height, err := terminal.GetSize(0)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get terminal size")
	}

	spin := spinner.New()
	spin.Style = spinnerStyle
	spin.Spinner = spinner.Meter
	spin.Tick()
	return &view{
		width:   width,
		height:  height,
		spinner: spin,
	}, nil
}

func (v *view) Init() tea.Cmd {
	// For initial IO
	// Could use to fetch data from the server
	// Could use to start the spinner, etc.
	return tick
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
		case "up", "k":
			v.viewport.LineUp(1)
			return v, nil
		case "down", "j":
			v.viewport.LineDown(1)
			return v, nil
		}
	case tea.WindowSizeMsg:
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height)
			v.viewport.SetContent(v.content)
			v.ready = true
		} else {
			v.width = msg.Width
			v.height = msg.Height
		}
	case tickMsg:
		v.spinner, _ = v.spinner.Update(v.spinner.Tick())
		return v, tick
	case errMsg:
		v.err = msg
		return v, nil
	}
	// Handle keyboard and mouse events in the viewport
	v.viewport, cmd = v.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

func (v *view) View() string {
	if v.err != nil {
		return v.err.Error()
	}

	if !v.ready {
		return "\n  Initializing..."
	}
	return v.viewport.View()
}

type tickMsg time.Time

func tick() tea.Msg {
	time.Sleep(time.Millisecond * 200)
	return tickMsg{}
}
