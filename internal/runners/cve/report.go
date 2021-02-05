package cve

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Report struct {
	proj *project.Project
	auth *authentication.Auth
	out  output.Outputer
}

type ReportParams struct {
	Namespace *project.Namespaced
}

func NewReport(prime primeable) *Report {
	return &Report{prime.Project(), prime.Auth(), prime.Output()}
}

type reportData struct {
	Project   string                    `json:"project"`
	CommitID  string                    `json:"commitID"`
	Date      time.Time                 `json:"generated_on"`
	Histogram []medmodel.SeverityCount  `json:"vulnerability_histogram"`
	Packages  []DetailedByPackageOutput `json:"packages"`
}

type reportDataPrinter struct {
	output output.Outputer
	data   *reportData
}

func (r *Report) Run(params *ReportParams) error {
	ns := params.Namespace
	if ns == nil {
		if r.proj == nil {
			return locale.NewInputError("err_no_project")
		}
		ns = r.proj.Namespace()
	}

	if !r.auth.Authenticated() {
		return errs.AddTips(
			locale.NewError("cve_needs_authentication"),
			locale.T("auth_tip"),
		)
	}

	resp, err := model.FetchProjectVulnerabilities(r.auth, ns.Owner, ns.Project)
	if err != nil {
		return locale.WrapError(err, "cve_mediator_resp", "Failed to retrieve vulnerability information")
	}

	var packageVulnerabilities []DetailedByPackageOutput
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

		cves := make(map[string][]medmodel.Vulnerability)
		for _, ve := range v.Vulnerabilities {
			if _, ok := cves[ve.Version]; !ok {
				cves[ve.Version] = []medmodel.Vulnerability{}
			}
			cves[ve.Version] = append(cves[ve.Version], ve)
		}

		for ver, vuls := range cves {
			packageVulnerabilities = append(packageVulnerabilities, DetailedByPackageOutput{
				v.Name, ver, vuls,
			})
		}
	}

	reportOutput := &reportData{
		Project:  resp.Project.Name,
		CommitID: resp.Project.Commit.CommitID,
		Date:     time.Now(),

		Histogram: resp.Project.Commit.VulnerabilityHistogram,
		Packages:  packageVulnerabilities,
	}

	rdp := &reportDataPrinter{
		r.out,
		reportOutput,
	}

	r.out.Print(rdp)

	return nil
}

func (od *reportDataPrinter) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return od.data
	}
	ri := &ReportInfo{
		fmt.Sprintf("[ACTIONABLE]%s[/RESET]", od.data.Project),
		od.data.CommitID,
		od.data.Date.Format("01/02/06"),
	}
	od.output.Print(struct {
		*ReportInfo `opts:"verticalTable"`
	}{ri})

	if len(od.data.Histogram) == 0 {
		od.output.Print("")
		od.output.Print(fmt.Sprintf("[SUCCESS]✔ %s[/RESET]", locale.Tl("no_cves", "No CVEs detected!")))

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
	for _, ap := range od.data.Packages {
		od.output.Print(fmt.Sprintf("[NOTICE]%s %s[/RESET]", ap.Name, ap.Version))
		od.output.Print(locale.Tl("report_package_vulnerabilities", "{{.V0}} Vulnerabilities", strconv.Itoa(len(ap.Details))))
		for i, d := range ap.Details {
			bar := "├─"
			if i == len(ap.Details)-1 {
				bar = "└─"
			}
			severity := d.Severity
			if severity == "CRITICAL" {
				severity = fmt.Sprintf("[ERROR]%s[/RESET]", severity)
			}
			od.output.Print(fmt.Sprintf("  %s %-12s [ACTIONABLE]%s[/RESET]", bar, severity, d.CveId))
		}
		od.output.Print("")
	}

	return output.Suppress
}
