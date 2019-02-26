package mock

import (
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/go-openapi/strfmt"
)

type RequesterOptions uint8

const NoOptions RequesterOptions = 0

const (
	NoArtifacts RequesterOptions = 1 << iota
	MultiArtifacts
	InvalidURL
)

type HeadchefRequesterMock struct {
	options RequesterOptions

	buildStarted   headchef.RequestBuildStarted
	buildFailed    headchef.RequestBuildFailed
	buildCompleted headchef.RequestBuildCompleted
	failure        headchef.RequestFailure
	close          headchef.RequestClose
}

func (r *HeadchefRequesterMock) OnBuildStarted(f headchef.RequestBuildStarted) {
	r.buildStarted = f
}

func (r *HeadchefRequesterMock) OnBuildFailed(f headchef.RequestBuildFailed) {
	r.buildFailed = f
}

func (r *HeadchefRequesterMock) OnBuildCompleted(f headchef.RequestBuildCompleted) {
	r.buildCompleted = f
}

func (r *HeadchefRequesterMock) OnFailure(f headchef.RequestFailure) {
	r.failure = f
}

func (r *HeadchefRequesterMock) OnClose(f headchef.RequestClose) {
	r.close = f
}

func (r *HeadchefRequesterMock) option(op RequesterOptions) bool {
	return r.options&op != 0
}

func (r *HeadchefRequesterMock) Start() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		r.buildStarted()
		artifacts := []*headchef_models.BuildCompletedArtifactsItems0{}
		if !r.option(NoArtifacts) {
			u := strfmt.URI("http://test.tld/archive.tar.gz")
			if r.option(InvalidURL) {
				u = strfmt.URI("htps;/not-a-url")
			}
			artifacts = append(artifacts, &headchef_models.BuildCompletedArtifactsItems0{
				URI: &u,
			})
			if r.option(MultiArtifacts) {
				artifacts = append(artifacts, artifacts[0])
			}
		}
		r.buildCompleted(headchef_models.BuildCompleted{
			Artifacts: artifacts,
		})
		r.close()
	}()
}

func NewHeadChefRequesterMock(opts RequesterOptions) *HeadchefRequesterMock {
	return &HeadchefRequesterMock{
		options:        opts,
		buildStarted:   func() {},
		buildFailed:    func(message string) {},
		buildCompleted: func(headchef_models.BuildCompleted) {},
		failure:        func(*failures.Failure) {},
		close:          func() {},
	}
}

type Mock struct {
}

func Init() *Mock {
	return &Mock{}
}

func (m *Mock) Close() {
}

func (m *Mock) Requester(opts RequesterOptions) headchef.InitRequester {
	return func(buildRequest *headchef_models.BuildRequest) headchef.Requester {
		return NewHeadChefRequesterMock(opts)
	}
}
