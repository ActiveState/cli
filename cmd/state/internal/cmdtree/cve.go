package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/cve"
	"github.com/ActiveState/cli/pkg/project"
)

func newCveCommand(prime *primer.Values) *captain.Command {
	runner := cve.NewCve(prime)
	params := cve.Params{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"security",
		locale.T("cve_title"),
		locale.T("cve_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("cve_namespace_arg"),
				Description: locale.T("cve_namespace_arg_description"),
				Value:       params.Namespace,
			},
		}, func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
	cmd.SetGroup(PlatformGroup)
	cmd.SetAliases("cve")
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)
	return cmd
}

// newReportCommand is a hidden, legacy alias of the parent command
func newReportCommand(prime *primer.Values) *captain.Command {
	report := cve.NewCve(prime)
	params := cve.Params{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"report",
		locale.T("cve_title"),
		locale.T("cve_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("cve_namespace_arg"),
				Description: locale.T("cve_namespace_arg_description"),
				Value:       params.Namespace,
			},
		}, func(_ *captain.Command, _ []string) error {
			return report.Run(&params)
		},
	)
	cmd.SetSupportsStructuredOutput()
	cmd.SetHidden(true)
	return cmd
}

func newOpenCommand(prime *primer.Values) *captain.Command {
	open := cve.NewOpen(prime)
	params := cve.OpenParams{}

	return captain.NewCommand(
		"open",
		locale.Tl("cve_open_title", "Opening Vulnerability Details Page"),
		locale.Tl("cve_open_cmd_description", "Open the given vulnerability details in your browser"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("cve_open_id_arg", "ID"),
				Description: locale.Tl("cve_open_id_arg_description", "The vulnerablility to open in your browser"),
				Value:       &params.ID,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return open.Run(params)
		},
	)
}
