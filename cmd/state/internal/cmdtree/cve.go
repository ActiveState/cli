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
		locale.Tl("cve_description", "Print project vulnerabilities"),
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
