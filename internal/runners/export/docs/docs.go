package docs

import (
	_ "embed"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/strutils"
)

type Docs struct {
	output output.Outputer
}

type Params struct{}

type primeable interface {
	primer.Outputer
}

func New(primer primeable) *Docs {
	return &Docs{primer.Output()}
}

//go:embed docs.md.tpl
var tpl string

func (d *Docs) Run(p *Params, cmd *captain.Command) error {
	stateCmd := cmd.TopParent()
	commands := make([][]*captain.Command, 0)
	commands = append(commands, grabChildren(stateCmd))

	var output string
	for _, cmds := range commands {
		out, err := strutils.ParseTemplate(tpl, map[string]interface{}{
			"Commands": cmds,
		}, nil)
		if err != nil {
			return errs.Wrap(err, "Could not parse template")
		}
		output += out
	}

	d.output.Print(output)

	return nil
}

func grabChildren(cmd *captain.Command) []*captain.Command {
	children := []*captain.Command{}
	for _, child := range cmd.Children() {
		if child.Hidden() {
			continue
		}
		children = append(children, child)
		children = append(children, grabChildren(child)...)
	}

	return children
}
