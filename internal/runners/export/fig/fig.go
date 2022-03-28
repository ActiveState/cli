package fig

import (
	_ "embed"
	"encoding/json"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/strutils"
)

type Fig struct {
	output output.Outputer
}

type Params struct{}

type primeable interface {
	primer.Outputer
}

type ExportCmd struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Options     []ExportOpt `json:"options"`
	Args        []ExportArg `json:"args"`
	SubCommands []ExportCmd `json:"subcommands"`
}

type ExportOpt struct {
	Name        []string    `json:"name"`
	Description string      `json:"description"`
	Args        []ExportArg `json:"args"`
}

type ExportArg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsOptional  bool   `json:"isOptional"`
}

func New(primer primeable) *Fig {
	return &Fig{primer.Output()}
}

//go:embed state.js.tpl
var tpl string

func (d *Fig) Run(p *Params, cmd *captain.Command) error {
	export := exportCmd(cmd.TopParent())
	v, err := json.MarshalIndent(export, "", "    ")
	if err != nil {
		return locale.WrapError(err, "", "Could not marshal command structure")
	}

	out, err := strutils.ParseTemplate(tpl, map[string]interface{}{
		"Export": string(v),
	})
	if err != nil {
		return locale.WrapError(err, "", "Could not parse template")
	}

	d.output.Print(out)

	return nil
}

func exportCmd(cmd *captain.Command) ExportCmd {
	export := ExportCmd{
		Name:        cmd.Name(),
		Description: cmd.Description(),
	}

	for _, f := range cmd.Flags() {
		opt := ExportOpt{
			Description: f.Description,
		}

		if f.Name != "" {
			opt.Name = append(opt.Name, "--"+f.Name)
		}

		if f.Shorthand != "" {
			opt.Name = append(opt.Name, "-"+f.Shorthand)
		}

		export.Options = append(export.Options, opt)
	}

	for _, a := range cmd.Arguments() {
		export.Args = append(export.Args, ExportArg{
			Name:        a.Name,
			Description: a.Description,
			IsOptional:  !a.Required,
		})
	}

	for _, c := range cmd.Children() {
		export.SubCommands = append(export.SubCommands, exportCmd(c))
	}

	return export
}
