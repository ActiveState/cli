package namespace

import (
	"fmt"
	"path/filepath"
)

var (
	namespaceExtension = "sock"
)

type Namespace struct {
	RootDir    string
	AppName    string
	AppVersion string
	AppHash    string
}

func (n *Namespace) String() string {
	filename := fmt.Sprintf(
		"%s-%s-%s.%s",
		n.AppName,
		n.AppVersion,
		n.AppHash,
		namespaceExtension,
	)

	return filepath.Join(n.RootDir, filename)
}
