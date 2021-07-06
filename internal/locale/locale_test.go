package locale

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	_ "github.com/ActiveState/cli/internal/config"
)

func TestInitAndT(t *testing.T) {
	translation := T("state_description")
	assert.NotZero(t, len(translation))
}

func TestGetLocalePath(t *testing.T) {
	path := getLocalePath()
	assert.Regexp(t, regexp.MustCompile(`[/locale/|\\locale\\]`), path, "Should detect locale path")
}

func TestGetLocaleFlag(t *testing.T) {
	args = []string{"--locale", "zz-ZZ"}
	flag := getLocaleFlag()
	assert.Equal(t, "zz-ZZ", flag, "Locale flag should be detected")
}

func TestSet(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	defer require.NoError(t, cfg.Close())
	err = Set("nl-NL")
	assert.NoError(t, err)
	assert.Equal(t, "nl-NL", cfg.GetString("Locale"), "Locale should be set to nl-NL")

	err = Set("en-US")
	assert.NoError(t, err)
	assert.Equal(t, "en-US", cfg.GetString("Locale"), "Locale should be set to en-US")

	err = Set("zz-ZZ")
	assert.Error(t, err)
	assert.NotEqual(t, "zz-ZZ", cfg.GetString("Locale"), "Locale should not be set to zz-ZZ")
}
