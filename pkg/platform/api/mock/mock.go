package mock

import (
	"fmt"
	"log"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/models"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String()),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockSignS3URI() {
	m.httpmock.Register("GET", "/s3/sign/http:%2F%2Ftest.tld%2Farchive.tar.gz")
}

func (m *Mock) MockVcsGetCheckpoint() {
	m.httpmock.Register("GET", "/vcs/commits/00010001-0001-0001-0001-000100010001/checkpoint")
}

func (m *Mock) MockVcsGetCheckpointPython() {
	m.MockVcsGetCheckpointCustomReq(&models.Checkpoint{
		Namespace:   "language",
		Requirement: "Python",
	})
}

func (m *Mock) MockVcsGetCheckpointCustomReq(requirement *models.Checkpoint) {
	jsonBytes, err := requirement.MarshalBinary()
	if err != nil {
		log.Panicf("Error during marshalling requirement: %v", err)
	}
	json := fmt.Sprintf("[%s]", string(jsonBytes))
	m.httpmock.RegisterWithResponseBody("GET", "/vcs/commits/00010001-0001-0001-0001-000100010001/checkpoint", 200, json)
}

func (m *Mock) MockGetProject() {
	m.httpmock.Register("GET", "/organizations/string/projects/string")
}

func (m *Mock) MockGetProjectDiffCommit() {
	m.httpmock.RegisterWithResponse("GET", "/organizations/string/projects/string", 200, "organizations/string/projects/string-diff-commit")
}

func (m *Mock) MockGetProject404() {
	m.httpmock.RegisterWithCode("GET", "/organizations/string/projects/string", 404)
}

func (m *Mock) MockGetOrganizations() {
	httpmock.RegisterWithCode("GET", "/organizations", 200)
}

func (m *Mock) MockGetOrganization() {
	httpmock.RegisterWithCode("GET", "/organizations/string", 200)
}

func (m *Mock) MockGetOrganization404() {
	httpmock.RegisterWithCode("GET", "/organizations/string", 404)
}
