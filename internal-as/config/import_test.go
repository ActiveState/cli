package config

import (
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/stretchr/testify/require"
)

func Test_importLegacyConfigFromBlob(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	err = config.importLegacyConfigFromBlob([]byte(`projects:
  ActiveState/cli:
  - /Users/nathanrijksen/Projects/cli
`))
	require.NoError(t, err, errs.JoinMessage(err))
}
