package distribution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObtain(t *testing.T) {
	dist, fail := Obtain()
	assert.NoError(t, fail.ToError(), "Should obtain distribution")
	assert.NotZero(t, dist.Languages, "Should return at least one language")
	assert.NotZero(t, dist.Artefacts, "Should return at least one artefact")
}
