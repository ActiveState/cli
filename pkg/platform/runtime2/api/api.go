package api

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	runtime "github.com/ActiveState/cli/pkg/platform/runtime2"
)

// ClientProvider is the interface for all functions that involve backend communication
type ClientProvider interface {
	Solve() (*inventory_models.Order, error)
	Build(*inventory_models.Order) (*BuildResult, error)
	BuildLog(msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (BuildLogger, error)
}

// BuildLogger is an interface to communicate with the build log streamer
type BuildLogger interface {
	Wait()
	Close()
	// BuiltArtifactChannel returns a channel of artifact IDs for artifacts that are ready to be downloaded.
	BuiltArtifactsChannel() chan runtime.ArtifactID
	Err() <-chan error
}

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine runtime.BuildEngine
	Recipe      *inventory_models.Recipe
}
