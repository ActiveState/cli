package svcctl

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/ipc"
)

func NewIPCNamespace() *ipc.Namespace {
	return &ipc.Namespace{
		RootDir:    filepath.Join(os.TempDir(), "state-test-ipc"),
		AppName:    "state",
		AppChannel: "default",
	}
}
