package mock

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/client"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

// Mock registers some common http requests usually used by the model
type Mock struct {
	httpmock   *httpmock.HTTPMock
	responders map[string]string
}

var mock *httpmock.HTTPMock

// Init initializes the mocking helper
func Init() *Mock {
	mock := &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceGraphQL).String()),
		map[string]string{},
	}
	mock.httpmock.RegisterWithResponder("POST", "", func(req *http.Request) (int, string) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return 500, err.Error()
		}
		for match, response := range mock.responders {
			var matchable bytes.Buffer
			if err := json.NewEncoder(&matchable).Encode(strings.TrimSpace(match)); err != nil {
				logging.Panic("Could not encode matchable: %v", err)
			}
			matchProcessed := strings.Trim(strings.TrimSpace(matchable.String()), `"`)
			if strings.Contains(string(body), matchProcessed) {
				return 200, response
			}
		}
		return 500, "No match found"
	})
	return mock
}

// Close de-activates the mocking helper
func (m *Mock) Close() {
	httpmock.DeActivate()
}

// Mock registers mocks for requests for receiving signed S3 URIs to packages
func (m *Mock) ProjectByOrgAndName() {
	m.responders[client.ProjectByOrgAndName().Query()] = "ProjectByOrgAndName"
}
