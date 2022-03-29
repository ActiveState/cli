package namespace

import (
	"fmt"
	"path/filepath"
	"strings"
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
		strings.ReplaceAll(n.AppChannel, "/", "--"),
		namespaceExtension,
	)

	return filepath.Join(n.RootDir, filename)
}
