package mock

import (
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type ResponseType int

const (
	Started ResponseType = iota
	Failed
	Completed
	RunFail
	RunFailMalformed
)

type ArtifactsOption string

const (
	Skip    ArtifactsOption = "-skip_artifacts"
	Invalid ArtifactsOption = "-invalid_artifacts"
	BadURI  ArtifactsOption = "-baduri_artifacts"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceHeadChef).String()),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockBuilds(respType ResponseType, artOpts ...ArtifactsOption) {
	regWithResp := m.httpmock.RegisterWithResponse
	regWithBody := m.httpmock.RegisterWithResponseBody

	path := "/v1/builds"

	switch respType {
	case Started:
		regWithResp("POST", path, 202, "builds-started")
	case Failed:
		regWithResp("POST", path, 201, "builds-failed")
	case Completed:
		var dir, suffix string
		if runtime.GOOS == "windows" {
			dir = "windows"
		}

		if hasOpt(artOpts, Invalid) {
			dir = ""
			suffix = string(Invalid)
		}

		if hasOpt(artOpts, BadURI) {
			suffix = string(BadURI)
		}

		if hasOpt(artOpts, Skip) {
			dir = ""
			suffix = string(Skip)
		}

		file := filepath.Join(dir, "builds-completed"+suffix)
		regWithResp("POST", path, 201, file)

	case RunFail:
		regWithBody("POST", path, 500, `{"message": "no"}`)
	case RunFailMalformed:
		regWithBody("POST", path, 201, `{"type": "no"}`)
	default:
		panic("use a valid ResponseType constant")
	}
}

func hasOpt(artOpts []ArtifactsOption, opt ArtifactsOption) bool {
	for _, artOpt := range artOpts {
		if artOpt == opt {
			return true
		}
	}
	return false
}
