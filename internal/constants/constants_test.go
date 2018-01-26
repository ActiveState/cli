package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
	funk "github.com/thoas/go-funk"
)

func TestConstants(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(true, funk.Contains(ConfigFileName, ConfigName), "ConfigFileName should contain ConfigName")
	assert.Equal(true, funk.Contains(ConfigFileName, ConfigFileType), "ConfigFileName should contain ConfigFileType")
}
