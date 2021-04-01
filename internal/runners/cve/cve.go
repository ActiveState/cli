package cve

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Cve struct {
	proj *project.Project
	auth *authentication.Auth
	out  output.Outputer
}

type outputData struct {
	Project   string                   `json:"project"`
	CommitID  string                   `json:"commitID"`
	Histogram []medmodel.SeverityCount `json:"vulnerability_histogram"`
	Packages  []ByPackageOutput        `json:"packages"`
}

type outputDataPrinter struct {
	output output.Outputer
	data   *outputData
}

type ByPackageOutput struct {
	Name     string `json:"name" locale:"state_cve_package_name,Name"`
	Version  string `json:"version" locale:"state_cve_package_version,Version"`
	CveCount int    `json:"cve_count" locale:"state_cve_package_count,Count"`
}

type ProjectInfo struct {
	Project  string `locale:"project,Project"`
	CommitID string `locale:"commit_id,Commit ID"`
}

type primeable interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
}

func NewCve(prime *primer.Values) *Cve {
	return &Cve{prime.Project(), prime.Auth(), prime.Output()}
}

func (c *Cve) Run() error {
	if c.proj == nil {
		return locale.NewError("cve_no_project", "No project found at the current directory.")
	}

	if !c.auth.Authenticated() {
		return errs.AddTips(
			locale.NewError("cve_needs_authentication"),
			locale.T("auth_tip"),
		)
	}

	resp, err := model.FetchCommitVulnerabilities(c.auth, c.proj.CommitID())
	if err != nil {
		return locale.WrapError(err, "cve_mediator_resp", "Failed to retrieve vulnerability information")
	}

	details := model.ExtractPackageVulnerabilities(resp.Sources)
	packageVulnerabilities := make([]ByPackageOutput, 0, len(details))
	for _, v := range details {
		packageVulnerabilities = append(packageVulnerabilities, ByPackageOutput{
			v.Name, v.Version, len(v.Details),
		})
	}

	cveOutput := &outputData{
		Project:   c.proj.Name(),
		CommitID:  resp.CommitID,
		Histogram: resp.VulnerabilityHistogram,
		Packages:  packageVulnerabilities,
	}

	odp := &outputDataPrinter{
		c.out,
		cveOutput,
	}

	c.out.Print(odp)

	return nil
}

type SeverityCountOutput struct {
	Count    string `locale:"count,Count"`
	Severity string `locale:"severity,Severity"`
}

func (od *outputDataPrinter) printFooter() {
	od.output.Print("")
	od.output.Print([]string{
		locale.Tl("cve_hint_report", "To view a detailed report for this runtime, run [ACTIONABLE]state security report[/RESET]"),
		locale.Tl("cve_hint_specific_report", "For a specific runtime, run [ACTIONABLE]state security report [Organization/Project][/RESET]"),
	})
}

func (od *outputDataPrinter) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return od.data
	}
	pi := &ProjectInfo{
		od.data.Project,
		od.data.CommitID,
	}
	od.output.Print(struct {
		*ProjectInfo `opts:"verticalTable"`
	}{pi})

	if len(od.data.Histogram) == 0 {
		od.output.Print("")
		od.output.Print(fmt.Sprintf("[SUCCESS]✔ %s[/RESET]", locale.Tl("no_cves", "No CVEs detected!")))
		od.printFooter()
		return output.Suppress
	}

	hist := make([]*SeverityCountOutput, 0, len(od.data.Histogram))
	totalCount := 0
	for _, h := range od.data.Histogram {
		totalCount += h.Count
		var ho *SeverityCountOutput
		if h.Severity == "CRITICAL" {
			ho = &SeverityCountOutput{
				fmt.Sprintf("[ERROR]%d[/RESET]", h.Count),
				fmt.Sprintf("[ERROR]%s[/RESET]", h.Severity),
			}
		} else {
			ho = &SeverityCountOutput{
				fmt.Sprintf("%d", h.Count),
				h.Severity,
			}
		}
		hist = append(hist, ho)
	}
	od.output.Print(output.Heading(fmt.Sprintf("%d Vulnerabilities", totalCount)))
	od.output.Print(hist)

	od.output.Print(output.Heading(fmt.Sprintf("%d Affected Packages", len(od.data.Packages))))
	od.output.Print(od.data.Packages)

	od.printFooter()
	return output.Suppress
}
