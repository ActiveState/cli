package distribution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestObtain(t *testing.T) {
	root := environment.GetRootPathUnsafe()
	err := os.Chdir(filepath.Join(root, "test"))
	assert.NoError(t, err, "Changed dir")

	dist, fail := Obtain()
	assert.NoError(t, fail.ToError(), "Should obtain distribution")
	assert.NotZero(t, dist.Languages, "Should return at least one language")
	assert.NotZero(t, dist.Artifacts, "Should return at least one artifact")
}
