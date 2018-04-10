package selfupdate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"self-update"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}
