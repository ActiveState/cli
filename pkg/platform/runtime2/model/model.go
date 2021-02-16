package model

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
)

// ClientProvider is the interface for all functions that involve backend communication
type ClientProvider interface {
	Solve() (*inventory_models.Order, error)
	Build(*inventory_models.Order) (*BuildResult, error)
	BuildLog(msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (BuildLog, error)
}

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine build.BuildEngine
	Recipe      *inventory_models.Recipe
}
