package donotshipme

import (
	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/primer"
)

func init() {
	if constants.ChannelName == "release" {
		panic("This file is for experimentation only, it should not be shipped as is. CmdTree is internal to the State command and should remain that way or be refactored.")
	}
}

func CmdTree(prime *primer.Values, args ...string) *cmdtree.CmdTree {
	return cmdtree.New(prime, args...)
}