package cve

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
}

type Cve struct {
	proj *project.Project
	auth *authentication.Auth
	out  output.Outputer
}

type CveInfo struct {
	Project  string `locale:"project,Project"`
	CommitID string `locale:"commit_id,Commit ID"`
	Date     string `locale:"generated_on,Generated on"`
}

func NewCve(prime primeable) *Cve {
	return &Cve{prime.Project(), prime.Auth(), prime.Output()}
}

type Params struct {
	Namespace *project.Namespaced
}

type cveData struct {
	Project   string                       `json:"project"`
	CommitID  string                       `json:"commitID"`
	Date      time.Time                    `json:"generated_on"`
	Histogram []medmodel.SeverityCount     `json:"vulnerability_histogram"`
	Packages  []model.PackageVulnerability `json:"packages"`
}

type cveOutput struct {
	output output.Outputer
	data   *cveData
}

func (r *Cve) Run(params *Params) error {
	if !params.Namespace.IsValid() && r.proj == nil {
		return rationalize.ErrNoProject
	}

	if !r.auth.Authenticated() {
		return errs.AddTips(
			locale.NewInputError("cve_needs_authentication"),
			locale.T("auth_tip"),
		)
	}

	vulnerabilities, err := r.fetchVulnerabilities(*params.Namespace)
	if err != nil {
		var errProjectNotFound *model.ErrProjectNotFound
		if errors.As(err, &errProjectNotFound) {
			return locale.WrapExternalError(err, "cve_mediator_resp_not_found", "That project was not found")
		}
		return locale.WrapError(err, "cve_mediator_resp", "Failed to retrieve vulnerability information")
	}

	packageVulnerabilities := model.ExtractPackageVulnerabilities(vulnerabilities.Sources)

	ns := params.Namespace
	if !ns.IsValid() {
		ns = r.proj.Namespace()
	}

	r.out.Print(&cveOutput{
		r.out,
		&cveData{
			Project:   ns.String(),
			CommitID:  vulnerabilities.CommitID,
			Date:      time.Now(),
			Histogram: vulnerabilities.VulnerabilityHistogram,
			Packages:  packageVulnerabilities,
		},
	})

	return nil
}

func (r *Cve) fetchVulnerabilities(namespaceOverride project.Namespaced) (*medmodel.CommitVulnerabilities, error) {
	if namespaceOverride.IsValid() && namespaceOverride.CommitID == nil {
		resp, err := model.FetchProjectVulnerabilities(r.auth, namespaceOverride.Owner, namespaceOverride.Project)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch vulnerability information for project %s", namespaceOverride.String())
		}
		return resp.Commit, nil
	}

	// fetch by commit ID
	var commitID string
	if namespaceOverride.IsValid() {
		commitID = namespaceOverride.CommitID.String()
	} else {
		var err error
		commitUUID, err := localcommit.Get(r.proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get local commit")
		}
		commitID = commitUUID.String()
	}
	resp, err := model.FetchCommitVulnerabilities(r.auth, commitID)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch vulnerability information for commit %s", commitID)
	}
	return resp, nil
}

type SeverityCountOutput struct {
	Count    string `locale:"count,Count" json:"count"`
	Severity string `locale:"severity,Severity" json:"severity"`
}

func (rd *cveOutput) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return rd.data
	}
	ri := &CveInfo{
		fmt.Sprintf("[ACTIONABLE]%s[/RESET]", rd.data.Project),
		rd.data.CommitID,
		rd.data.Date.Format("01/02/06"),
	}
	rd.output.Print(struct {
		*CveInfo `opts:"verticalTable"`
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
	rd.output.Print(output.Title(fmt.Sprintf("%d Vulnerabilities", totalCount)))
	rd.output.Print(hist)

	rd.output.Print(output.Title(fmt.Sprintf("%d Affected Packages", len(rd.data.Packages))))
	for _, ap := range rd.data.Packages {
		rd.output.Print(fmt.Sprintf("[NOTICE]%s %s[/RESET]", ap.Name, ap.Version))
		rd.output.Print(locale.Tl("cve_package_vulnerabilities", "{{.V0}} Vulnerabilities", strconv.Itoa(len(ap.Details))))

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
			bar := output.TreeMid
			if i == len(ap.Details)-1 {
				bar = output.TreeEnd
			}
			severity := d.Severity
			if severity == "CRITICAL" {
				severity = fmt.Sprintf("[ERROR]%-10s[/RESET]", severity)
			}
			rd.output.Print(fmt.Sprintf("  %s %-10s [ACTIONABLE]%s[/RESET]", bar, severity, d.CveID))
		}
		rd.output.Print("")
	}

	rd.output.Print("")
	rd.output.Print([]string{
		locale.Tl("cve_hint_cve", "To view a specific CVE, run [ACTIONABLE]state security open [cve-id][/RESET]."),
	})
	return output.Suppress
}

func (rd *cveOutput) MarshalStructured(format output.Format) interface{} {
	return rd.data
}
