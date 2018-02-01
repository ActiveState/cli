package locale

import (
	"testing"

	_ "github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitAndT(t *testing.T) {
	translation := T("state_description")
	assert.NotZero(t, len(translation))

	translation = Tt("usage_tpl")
	assert.Contains(t, translation, "{{", "Translation should contain template tags")
}

func TestGetLocalePath(t *testing.T) {
	path := getLocalePath()
	assert.Contains(t, path, constants.LibraryNamespace+constants.LibraryName+"/locale/", "Should detect locale path")
}

func TestGetLocaleFlag(t *testing.T) {
	args = []string{"--locale", "zz-ZZ"}
	flag := getLocaleFlag()
	assert.Equal(t, "zz-ZZ", flag, "Locale flag should be detected")
}

func TestSet(t *testing.T) {
	Set("nl-NL")
	assert.Equal(t, "nl-NL", viper.GetString("Locale"), "Locale should be set to nl-NL")

	Set("en-US")
	assert.Equal(t, "en-US", viper.GetString("Locale"), "Locale should be set to en-US")

	exitCode := 0
	exit = func(code int) {
		exitCode = 1
	}

	Set("zz-ZZ")
	assert.Equal(t, 1, exitCode, "Should not be able to set nonexistant language")
	assert.NotEqual(t, "zz-ZZ", viper.GetString("Locale"), "Locale should not be set to zz-ZZ")
}
