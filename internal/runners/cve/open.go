package cve

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/skratchdot/open-golang/open"
)

const cveURLPrefix = "https://nvd.nist.gov/vuln/detail/"

type Open struct {
	out output.Outputer
}

type OpenParams struct {
	ID string
}

func NewOpen(prime primeable) *Open {
	return &Open{
		out: prime.Output(),
	}
}

func (o *Open) Run(params OpenParams) error {
	cveURL := fmt.Sprintf("%s%s", cveURLPrefix, params.ID)
	err := open.Run(cveURL)
	if err != nil {
		return errs.AddTips(
			locale.WrapError(err, "cve_open_url_err", "Could not open CVE detail URL: {{.V0}}", cveURL),
			locale.Tr("browser_fallback", "vulnerability details", cveURL),
		)
	}
	o.out.Print(locale.Tl("cve_open_url", "Vulnerability detail URL: [ACTIONABLE]{{.V0}}[/RESET]", cveURL))

	return nil
}
