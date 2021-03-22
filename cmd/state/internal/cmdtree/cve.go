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

	cmd := captain.NewCommand(
		"security",
		locale.Tl("cve_title", "Vulnerability Summary"),
		locale.Tl("cve_description", "Show a summary of project vulnerabilities"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
	cmd.SetGroup(PlatformGroup)
	cmd.SetAliases("cve")
	return cmd
}

func newReportCommand(prime *primer.Values) *captain.Command {
	report := cve.NewReport(prime)
	params := cve.ReportParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"report",
		locale.Tl("cve_report_title", "Vulnerability Report"),
		locale.Tl("cve_report_cmd_description", "Show a detailed report of project vulnerabilities"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("cve_report_namespace_arg", "Organization/Project"),
				Description: locale.Tl("cve_report_namespace_arg_description", "The project for which the report is created"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return report.Run(&params)
		},
	)
}

func newOpenCommand(prime *primer.Values) *captain.Command {
	open := cve.NewOpen(prime)
	params := cve.OpenParams{}

	return captain.NewCommand(
		"open",
		locale.Tl("cve_open_title", "Opening Vulnerability Details Page"),
		locale.Tl("cve_open_cmd_description", "Open the given vulnerability details in your browser"),
		prime.Output(),
		prime.Config(),
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
