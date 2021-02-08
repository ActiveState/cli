package cve

import (
	"fmt"
	"sort"
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

type ReportInfo struct {
	Project  string `locale:"project,Project"`
	CommitID string `locale:"commit_id,Commit ID"`
	Date     string `locale:"generated_on,Generated on"`
}

func NewReport(prime primeable) *Report {
	return &Report{prime.Project(), prime.Auth(), prime.Output()}
}

type DetailedByPackageOutput struct {
	Name    string                   `json:"name"`
	Version string                   `json:"version"`
	Details []medmodel.Vulnerability `json:"cves"`
}

type ReportParams struct {
	Namespace *project.Namespaced
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
		Project:  ns.String(),
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

func (rd *reportDataPrinter) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return rd.data
	}
	ri := &ReportInfo{
		fmt.Sprintf("[ACTIONABLE]%s[/RESET]", rd.data.Project),
		rd.data.CommitID,
		rd.data.Date.Format("01/02/06"),
	}
	rd.output.Print(struct {
		*ReportInfo `opts:"verticalTable"`
	}{ri})

	if len(rd.data.Histogram) == 0 {
		rd.output.Print("")
		rd.output.Print(fmt.Sprintf("[SUCCESS]✔ %s[/RESET]", locale.Tl("no_cves", "No CVEs detected!")))

		return output.Suppress
	}

	hist := make([]*SeverityCountOutput, 0, len(rd.data.Histogram))
	totalCount := 0
	for _, h := range rd.data.Histogram {
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
	rd.output.Print(output.Heading(fmt.Sprintf("%d Vulnerabilities", totalCount)))
	rd.output.Print(hist)

	rd.output.Print(output.Heading(fmt.Sprintf("%d Affected Packages", len(rd.data.Packages))))
	for _, ap := range rd.data.Packages {
		rd.output.Print(fmt.Sprintf("[NOTICE]%s %s[/RESET]", ap.Name, ap.Version))
		rd.output.Print(locale.Tl("report_package_vulnerabilities", "{{.V0}} Vulnerabilities", strconv.Itoa(len(ap.Details))))

		sort.SliceStable(ap.Details, func(i, j int) bool {
			sevI := ap.Details[i].Severity
			sevJ := ap.Details[j].Severity
			si := medmodel.ParseSeverityIndex(sevI)
			sj := medmodel.ParseSeverityIndex(sevJ)
			if si < sj {
				return true
			}
			if si == sj {
				return sevI < sevJ
			}
			return false
		})

		for i, d := range ap.Details {
			bar := "├─"
			if i == len(ap.Details)-1 {
				bar = "└─"
			}
			severity := d.Severity
			if severity == "CRITICAL" {
				severity = fmt.Sprintf("[ERROR]%-10s[/RESET]", severity)
			}
			rd.output.Print(fmt.Sprintf("  %s %-10s [ACTIONABLE]%s[/RESET]", bar, severity, d.CveId))
		}
		rd.output.Print("")
	}

	rd.output.Print("")
	rd.output.Print([]string{
		locale.Tl("cve_report_hint_cve", "To view a specific CVE, run [ACTIONABLE]state cve open [cve-id][/RESET]."),
	})
	return output.Suppress
}
