package request

type LocalProjectsRequest struct{}

func NewLocalProjectsRequest() *LocalProjectsRequest {
	return &LocalProjectsRequest{}
}

func (l *LocalProjectsRequest) Query() string {
	return `query {
		projects {
			namespace
			locations
		}
	}`
}

func (l *LocalProjectsRequest) Vars() map[string]interface{} {
	return nil
}
