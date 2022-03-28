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
	AppChannel string
}

func (n *Namespace) String() string {
	filename := fmt.Sprintf(
		"%s-%s.%s",
		n.AppName,
		n.AppChannel,
		namespaceExtension,
	)

	return filepath.Join(n.RootDir, filename)
}
