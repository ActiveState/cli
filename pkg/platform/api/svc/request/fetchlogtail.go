package request

import "github.com/ActiveState/cli/internal/gqlclient"

type FetchLogTail struct {
	gqlclient.RequestBase
}

func NewFetchLogTail() *FetchLogTail {
	return &FetchLogTail{}
}

func (r *FetchLogTail) Query() string {
	return `query() {
		fetchLogTail
	}`
}

func (r *FetchLogTail) Vars() (map[string]interface{}, error) {
	return nil, nil
}
