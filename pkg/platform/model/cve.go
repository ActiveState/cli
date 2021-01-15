package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/mediator"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchProjectVulnerabilities returns the vulnerability information of a project
func FetchProjectVulnerabilities(a *authentication.Auth, org, project string) (*model.ProjectVulnerabilities, error) {
	// This should be removed by
	if !a.Authenticated() {
		return nil, errs.AddTips(
			locale.NewError("cve_needs_authentication", "You need to be authenticated in order to access vulnerability information about your project."),
			locale.Tl("auth_tip", "Run `state auth` to authenticate."),
		)
	}
	req := request.VulnerabilitiesByProject(org, project)
	var resp model.ProjectVulnerabilities
	med := mediator.Get()
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run mediator request.")
	}

	msg := resp.Project.Message
	if msg != nil {
		return nil, locale.NewError("project_vulnerability_err", "Request to retrieve vulnerability information failed with error: {{.V0}}", *msg)
	}

	return &resp, nil
}
