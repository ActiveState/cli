package client

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
)

// var _ model.ClientProvider = &Default{}

// Default is the default client that actually talks to the backend
type Default struct{}

// NewDefault is the constructor for the Default client
func NewDefault() *Default {
	return &Default{}
}

func (d *Default) Solve() (*inventory_models.Order, error) {
	panic("implement me")
}

func (d *Default) Build(order *inventory_models.Order) (*model.BuildResult, error) {
	panic("implement me")
}

func (d *Default) BuildLog(msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (model.BuildLog, error) {
	panic("implement me")
}
