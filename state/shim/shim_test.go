package shim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShim(t *testing.T) {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"whoami"})

	err := Command.Execute()
	require.NoError(t, err)
}
