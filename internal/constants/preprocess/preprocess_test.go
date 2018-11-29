package preprocess

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	os.Setenv("APIENV", "preprocess_testing")
	for k, v := range Constants {
		assert.NotEmpty(t, v(), "Value for "+k+" is generated")
	}
}
