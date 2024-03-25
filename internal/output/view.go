package output

import (
	"os"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	leftPad        = 2
	verticalMargin = 7
	scrollUp       = "up"
	scrollDown     = "down"
)

type ContentProcessor interface {
	Content() string
}

type view struct {
	width         int
	height        int
	ready         bool
	footerMessage string
	viewport      viewport.Model
	processor     ContentProcessor
}

func NewView(out Outputer, processor ContentProcessor) (*view, error) {
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
		width:     width,
		height:    height,
		processor: processor,
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
			v.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			v.viewport.SetContent(v.processor.Content())
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

func (v *view) setFooterMessage(msg string) {
	v.footerMessage = msg
}

func (v *view) footerView() string {
	var footerText string
	scrollValue := v.viewport.ScrollPercent() * 100
	footerText += locale.Tl("footer_scroll", "... {{.V0}}% scrolled, use arrow and page keys to scroll. Press Q to quit.", strconv.Itoa(int(scrollValue)))
	footerText += v.footerMessage
	return lipgloss.NewStyle().Render(footerText)
}
