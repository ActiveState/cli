package socket

import (
	"fmt"
	"path/filepath"
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
