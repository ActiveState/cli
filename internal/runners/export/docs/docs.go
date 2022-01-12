package docs

import (
	_ "embed"

	"github.com/ActiveState/cli/internal/captain"
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
	children := grabChildren(stateCmd)

	out, err := strutils.ParseTemplate(tpl, map[string]interface{}{
		"Commands": children,
	})
	if err != nil {
		return err
	}

	d.output.Print(out)

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
