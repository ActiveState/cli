package client

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
)

// var _ model.ClientProvider = &Default{}

// Mock mocks an api client.  It can be used in tests to return configurable responses.
type Mock struct{}

// NewMock constructs a Test client
// TODO: It's responses are fully configurable and deterministic
func NewMock() *Mock {
	return &Mock{}
}

func (tc *Mock) Solve() (*inventory_models.Order, error) {
	panic("implement me")
}

func (tc *Mock) Build(order *inventory_models.Order) (*model.BuildResult, error) {
	panic("implement me")
}

// BuildLog returns a mocked BuildLogger implementation
func (tc *Mock) BuildLog(msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (model.BuildLogger, error) {
	panic("implement me")
}
