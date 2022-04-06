package svcctl

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/ipc"
)

func NewIPCSockPath() *ipc.SockPath {
	return &ipc.SockPath{
		RootDir:    filepath.Join(os.TempDir(), "state-test-ipc"),
		AppName:    "state",
		AppChannel: "default",
	}
}
