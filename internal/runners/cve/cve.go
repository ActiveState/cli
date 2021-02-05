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

type DetailedByPackageOutput struct {
	Name    string                   `json:"name"`
	Version string                   `json:"version"`
	Details []medmodel.Vulnerability `json:"cves"`
}

type ProjectInfo struct {
	Project  string `locale:"project,Project"`
	CommitID string `locale:"commit_id,Commit ID"`
}

type ReportInfo struct {
	Project  string `locale:"project,Project"`
	CommitID string `locale:"commit_id,Commit ID"`
	Date     string `locale:"generated_on,Generated on"`
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
			locale.NewError("cve_needs_authentication", "You need to be authenticated in order to access vulnerability information about your project."),
			locale.Tl("auth_tip", "Run `state auth` to authenticate."),
		)
	}

	resp, err := model.FetchProjectVulnerabilities(c.auth, c.proj.Owner(), c.proj.Name())
	if err != nil {
		return locale.WrapError(err, "cve_mediator_resp", "Failed to retrieve vulnerability information")
	}

	var packageVulnerabilities []ByPackageOutput
	visited := make(map[string]struct{})
	for _, v := range resp.Project.Commit.Ingredients {
		if len(v.Vulnerabilities) == 0 {
			continue
		}

		// Remove this block with story https://www.pivotaltracker.com/story/show/176508772
		// filter double entries
		if _, ok := visited[v.Name]; ok {
			continue
		}
		visited[v.Name] = struct{}{}

		countByVersion := make(map[string]int)
		for _, ve := range v.Vulnerabilities {
			if _, ok := countByVersion[ve.Version]; !ok {
				countByVersion[ve.Version] = 0
			}
			countByVersion[ve.Version]++
		}

		for ver, count := range countByVersion {
			packageVulnerabilities = append(packageVulnerabilities, ByPackageOutput{
				v.Name, ver, count,
			})
		}
	}

	cveOutput := &outputData{
		Project:   resp.Project.Name,
		CommitID:  resp.Project.Commit.CommitID,
		Histogram: resp.Project.Commit.VulnerabilityHistogram,
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
		locale.Tl("cve_hint_report", "To view a detailed report for this runtime, run [ACTIONABLE]state cve report[/RESET]"),
		locale.Tl("cve_hint_specific_report", "For a specific runtime, run [ACTIONABLE]state cve report [orgName/projectName][/RESET]"),
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
