package locale

import (
	"regexp"
	"testing"

	_ "github.com/ActiveState/cli/internal/config"
	"github.com/stretchr/testify/assert"
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


