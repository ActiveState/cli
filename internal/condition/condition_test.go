package condition

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_BuiltViaCI(t *testing.T) {
	t.Log("If you aren't running this on CI you can safely ignore this test failing")

	require.True(t, BuiltViaCI())
}

func Test_InUnitTest(t *testing.T) {
	require.True(t, InUnitTest())
}
