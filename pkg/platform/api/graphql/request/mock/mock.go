package mock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type Options uint8

const (
	NoOptions Options = iota
	Once
)

type Responder struct {
	matchable    string
	responseFile string
	options      Options
}

func NewResponder(match string, responseFile string, options Options) *Responder {
	var matchable bytes.Buffer
	if err := json.NewEncoder(&matchable).Encode(strings.TrimSpace(match)); err != nil {
		logging.Panic("Could not encode matchable: %v", err)
	}
	matchProcessed := strings.Trim(strings.TrimSpace(matchable.String()), `"`)
	return &Responder{matchProcessed, responseFile, options}
}

func (r *Responder) option(op Options) bool {
	return r.options&op != 0
}

// Mock registers some common http requests usually used by the model
type Mock struct {
	httpmock   *httpmock.HTTPMock
	responders []*Responder
}

// Init initializes the mocking helper
func Init() *Mock {
	mock := &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceGraphQL).String()),
		[]*Responder{},
	}
	mock.httpmock.RegisterWithResponder("POST", "", mock.handleRequest)
	return mock
}

// Reset unsets any responders, useful since this mock is special since it doesn't mock based on path
func (m *Mock) Reset() {
	m.responders = []*Responder{}
}

// Close de-activates the mocking helper
func (m *Mock) Close() {
	httpmock.DeActivate()
}

// Close de-activates the mocking helper
func (m *Mock) handleRequest(req *http.Request) (int, string) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return 500, err.Error()
	}
	for i, responder := range m.responders {
		if strings.Contains(string(body), responder.matchable) {
			if responder.option(Once) {
				// Delete responder
				m.responders = append(m.responders[:i], m.responders[i+1:]...)
			}
			return 200, responder.responseFile
		}
	}
	logging.Panic("No match found for request: %s, body: %s", req.URL.String(), string(body))
	return 500, ""
}

func (m *Mock) NoProjects(options Options) {
	m.responders = append(m.responders, NewResponder(request.ProjectByOrgAndName("", "").Query(), "NoProjects", options))
}

func (m *Mock) ProjectByOrgAndName(options Options) {
	m.responders = append(m.responders, NewResponder(request.ProjectByOrgAndName("", "").Query(), "Project", options))
}

func (m *Mock) ProjectByOrgAndNameNoCommits(options Options) {
	m.responders = append(m.responders, NewResponder(request.ProjectByOrgAndName("", "").Query(), "ProjectNoCommits", options))
}

func (m *Mock) Checkpoint(options Options) {
	m.responders = append(m.responders, NewResponder(request.CheckpointByCommit("").Query(), "Checkpoint", options))
}

func (m *Mock) CheckpointWithPrePlatform(options Options) {
	m.responders = append(m.responders, NewResponder(request.CheckpointByCommit("").Query(), "CheckpointPrePlatform", options))
}

func (m *Mock) NoCheckpoint(options Options) {
	m.responders = append(m.responders, NewResponder(request.CheckpointByCommit("").Query(), "NoCheckpoint", options))
}
