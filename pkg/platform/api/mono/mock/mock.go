package mock

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// Mock registers some common http requests usually used by the model
type Mock struct {
	httpmock *httpmock.HTTPMock
}

var mock *httpmock.HTTPMock

// Init initializes the mocking helper
func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceMono).String()),
	}
}

// Close de-activates the mocking helper
func (m *Mock) Close() {
	httpmock.DeActivate()
}

// MockSignS3URI registers mocks for requests for receiving signed S3 URIs to packages
func (m *Mock) MockSignS3URI() {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	m.httpmock.RegisterWithResponse("GET", "/s3/sign/http:%2F%2Ftest.tld%2Fpython"+ext, 200, "s3/sign/python"+ext+".json")
	m.httpmock.RegisterWithResponse("GET", "/s3/sign/http:%2F%2Ftest.tld%2Flegacy-python"+ext, 200, "s3/sign/legacy-python"+ext+".json")
}

// MockVcsGetCheckpoint registers mocks for version control commits
func (m *Mock) MockVcsGetCheckpoint() {
	m.httpmock.Register("GET", "/vcs/commits/00010001-0001-0001-0001-000100010001/checkpoint")
	m.httpmock.Register("GET", "/vcs/commits/00020002-0002-0002-0002-000200020002/checkpoint")
}

// MockVcsGetCheckpointPython registers a mock returning a VCS checkpoint for python
func (m *Mock) MockVcsGetCheckpointPython() {
	m.MockVcsGetCheckpointCustomReq(&mono_models.Checkpoint{
		Namespace:   "language",
		Requirement: "Python",
	})
}

// MockVcsGetCheckpointCustomReq registers a mock returning the platform data for a specific requirement at a given VCS checkpoint
func (m *Mock) MockVcsGetCheckpointCustomReq(requirement *mono_models.Checkpoint) {
	jsonBytes, err := requirement.MarshalBinary()
	if err != nil {
		log.Panicf("Error during marshalling requirement: %v", err)
	}
	json := fmt.Sprintf("[%s]", string(jsonBytes))
	m.httpmock.RegisterWithResponseBody("GET", "/vcs/commits/00010001-0001-0001-0001-000100010001/checkpoint", 200, json)
}

// MockGetProject registers mocks for project "string" and a VCS checkpoint
func (m *Mock) MockGetProject() {
	m.httpmock.Register("GET", "/organizations/string/projects/string")
	m.httpmock.Register("GET", "/vcs/commits/00010001-0001-0001-0001-000100010001/checkpoint")
}

// MockGetProjectNoLanguage returns a mock returning a project without a language set
func (m *Mock) MockGetProjectNoLanguage() {
	m.httpmock.RegisterWithResponse("GET", "/organizations/string/projects/string", 200, "organizations/string/projects/string-no-language")
}

// MockGetProjectDiffCommit registers a mock returning a project with a commit history
func (m *Mock) MockGetProjectDiffCommit() {
	m.httpmock.RegisterWithResponse("GET", "/organizations/string/projects/string", 200, "organizations/string/projects/string-diff-commit")
}

// MockGetProject404 registers a mock for a request for a non-existent project
func (m *Mock) MockGetProject404() {
	m.httpmock.RegisterWithCode("GET", "/organizations/string/projects/string", 404)
}

// MockGetOrganizations registers a mock returning organizations
func (m *Mock) MockGetOrganizations() {
	httpmock.RegisterWithCode("GET", "/organizations", 200)
}

// MockGetOrganization registers a mock returning the specific organization "string"
func (m *Mock) MockGetOrganization() {
	httpmock.RegisterWithCode("GET", "/organizations/string", 200)
}

// MockGetOrganization401 registers a mock for an organization request when we are not authenticated
func (m *Mock) MockGetOrganization401() {
	httpmock.RegisterWithCode("GET", "/organizations/string", 401)
}

// MockGetOrganizationLimits registers a mock returning the limits for an organization
func (m *Mock) MockGetOrganizationLimits() {
	httpmock.RegisterWithCode("GET", "/organizations/string/limits", 200)
}

// MockGetOrganizationLimitsReached registers a mock returning the limits for an organization that has reached its maximum users count
func (m *Mock) MockGetOrganizationLimitsReached() {
	httpmock.RegisterWithResponse("GET", "/organizations/string/limits", 200, "/organizations/string/limits-reached")
}

// MockInviteUser registers a mock for a request inviting a new user by email.
func (m *Mock) MockInviteUser() {
	httpmock.Register("POST", "/organizations/string/invitations/foo@bar.com")
}

// MockGetOrganizationLimits401 registers a mock for a limit request when we are not authenticated
func (m *Mock) MockGetOrganizationLimits401() {
	httpmock.RegisterWithCode("GET", "/organizations/string/limits", 401)
}

// MockGetOrganizationLimits403 registers a mock for a limit request that is forbidden due to missing user permissions
func (m *Mock) MockGetOrganizationLimits403() {
	httpmock.RegisterWithCode("GET", "/organizations/string/limits", 403)
}

// MockGetOrganizationLimits404 registers a mock for a limit request for a non-existent organization
func (m *Mock) MockGetOrganizationLimits404() {
	httpmock.RegisterWithCode("GET", "/organizations/string/limits", 404)
}

// MockGetOrganization404 registers a mock for a request for a non-existent organization
func (m *Mock) MockGetOrganization404() {
	httpmock.RegisterWithCode("GET", "/organizations/string", 404)
}

// MockInviteUserToOrg registers a mock invite request
func (m *Mock) MockInviteUserToOrg() {
	httpmock.RegisterWithCode("POST", "/organizations/string/invitations/foo@bar.com", 200)
}

// MockInviteUserToOrg404 registers a mock invite request for a non-existent organization
func (m *Mock) MockInviteUserToOrg404() {
	httpmock.RegisterWithCode("POST", "/organizations/string/invitations/string", 404)
}

// MockCommit registers a mock for a VCS commit
func (m *Mock) MockCommit() {
	m.httpmock.Register("POST", "/vcs/commit")
}
