package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/cve"
)

func newCveCommand(prime *primer.Values) *captain.Command {
	runner := cve.NewCve(prime)

	cmd := captain.NewCommand(
		"cve",
		locale.Tl("cve_title", "CVE Summary"),
		locale.Tl("cve_description", "Show a summary of CVE vulnerabilities"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)
	cmd.SetGroup(PlatformGroup)
	return cmd
}

func newReportCommand(prime *primer.Values) *captain.Command {
	report := cve.NewReport(prime)
	params := cve.ReportParams{}

	return captain.NewCommand(
		"report",
		locale.Tl("cve_report_title", "Print a vulnerability report"),
		locale.Tl("cve_report_cmd_description", "Print a vulnerability report"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("cve_report_namespace_arg", "organization/project"),
				Description: locale.Tl("cve_report_namespace_arg_description", "The project for which the report is created"),
				Value:       &params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return report.Run(&params)
		},
	)
}
