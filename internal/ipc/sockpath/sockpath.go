package sockpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

var (
	sockpathExtension = "sock"
)

type SockPath struct {
	RootDir    string
	AppName    string
	AppChannel string
}

func (n *SockPath) String() string {
	filename := fmt.Sprintf(
		"%s-%s.%s",
		n.AppName,
		strings.ReplaceAll(n.AppChannel, "/", "--"),
		sockpathExtension,
	)

	return filepath.Join(n.RootDir, filename)
}
