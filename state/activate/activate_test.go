package activate

import (
	"os"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	config.Init()
	locale.Init()
	code := m.Run()
	os.Exit(code)
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"activate"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}
