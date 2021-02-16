package runtime

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime2/impl"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model/client"
	"github.com/ActiveState/cli/pkg/project"
)

// TestSetup
func TestSetup(t *testing.T) {
	var mockProject *project.Project
	var mockMessageHandler impl.MessageHandler
	mockClient := client.NewMock()
	// specify behavior of mock client here.
	// ...

	s := NewSetupWithAPI(mockProject, mockMessageHandler, mockClient)

	s.InstallRuntime()
	// TODO: check error

	// TODO: check messageHandler calls

	r, _ := s.InstalledRuntime()
	// TODO: check runtime works
	r.Environ()
}
