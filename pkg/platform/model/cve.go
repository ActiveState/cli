package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/mediator"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Vulnerability model.Vulnerability
type PackageVulnerability struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Details []Vulnerability `json:"cves"`
}

// FetchProjectVulnerabilities returns the vulnerability information of a project
func FetchProjectVulnerabilities(auth *authentication.Auth, org, project string) (*model.ProjectVulnerabilities, error) {
	// This should be removed by https://www.pivotaltracker.com/story/show/176508740
	if !auth.Authenticated() {
		return nil, errs.AddTips(
			locale.NewError("cve_needs_authentication", "You need to be authenticated in order to access vulnerability information about your project."),
			locale.Tl("auth_tip", "Run `state auth` to authenticate."),
		)
	}
	req := request.VulnerabilitiesByProject(org, project)
	var resp model.ProjectResponse
	med := mediator.New(auth)
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run mediator request.")
	}

	msg := resp.ProjectVulnerabilities.Message
	if msg != nil {
		return nil, locale.NewError("project_vulnerability_err", "Request to retrieve vulnerability information failed with error: {{.V0}}", *msg)
	}

	return &resp.ProjectVulnerabilities, nil
}

// FetchCommitVulnerabilities returns the vulnerability information of a project
func FetchCommitVulnerabilities(auth *authentication.Auth, commitID string) (*model.CommitVulnerabilities, error) {
	// This should be removed by https://www.pivotaltracker.com/story/show/176508740
	if !auth.Authenticated() {
		return nil, errs.AddTips(
			locale.NewError("cve_needs_authentication", "You need to be authenticated in order to access vulnerability information about your project."),
			locale.Tl("auth_tip", "Run `state auth` to authenticate."),
		)
	}
	req := request.VulnerabilitiesByCommit(commitID)
	var resp model.CommitResponse
	med := mediator.New(auth)
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run mediator request.")
	}

	msg := resp.Message
	if msg != nil {
		return nil, locale.NewError("commit_vulnerability_err", "Request to retrieve vulnerability information for commit {{.V0}} failed with error: {{.V1}}", commitID, *msg)
	}

	return &resp.CommitVulnerabilities, nil
}

func ExtractPackageVulnerabilities(sources []model.SourceVulnerability) []PackageVulnerability {
	var packageVulnerabilities []PackageVulnerability
	visited := make(map[string]struct{})
	for _, v := range sources {
		if len(v.Vulnerabilities) == 0 {
			continue
		}

		// Remove this block with story https://www.pivotaltracker.com/story/show/176508772
		// filter double entries
		if _, ok := visited[v.Name]; ok {
			continue
		}
		visited[v.Name] = struct{}{}

		vuls := make([]Vulnerability, 0, len(v.Vulnerabilities))
		for _, ve := range v.Vulnerabilities {
			vuls = append(vuls, Vulnerability(ve))
		}
		packageVulnerabilities = append(packageVulnerabilities, PackageVulnerability{
			v.Name, v.Version, vuls,
		})
	}

	return packageVulnerabilities
}
