package packages

import (
	"strings"

	"github.com/ActiveState/cli/pkg/platform/model"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type view struct {
	packages []*model.IngredientAndVersion
	err      error
}

func NewView() *view {
	return &view{}
}

func (v *view) Init() tea.Cmd {
	// For initial IO
	// Could use to fetch data from the server
	// Could use to start the spinner, etc.
	return nil
}

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		stringMsg := strings.ToLower(msg.String())
		if stringMsg == "q" || stringMsg == "ctrl+c" {
			return v, tea.Quit
		}
	case errMsg:
		v.err = msg
		return v, nil
	}
	return v, nil
}

func (v *view) View() string {
	if v.err != nil {
		return v.err.Error()
	}
	return "Hello, world!"
}
