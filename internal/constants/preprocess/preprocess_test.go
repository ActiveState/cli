package preprocess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	for k, v := range Constants {
		assert.NotEmpty(t, v(), "Value for "+k+" is generated")
	}
}
