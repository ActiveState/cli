package sockpath

import (
	"fmt"
	"path/filepath"
	"strings"
)

var (
	sockpathExtension = "sock"
	maxChannelLength  = 12
)

type SockPath struct {
	RootDir    string
	AppName    string
	AppChannel string
}

func (n *SockPath) String() string {
	appChannel := strings.ReplaceAll(n.AppChannel, "/", "_")
	cStart := len(appChannel) - maxChannelLength
	if cStart < 0 {
		cStart = 0
	}
	appChannel = appChannel[cStart:]

	filename := fmt.Sprintf("%s-%s.%s", n.AppName, appChannel, sockpathExtension)

	return filepath.Join(n.RootDir, filename)
}
