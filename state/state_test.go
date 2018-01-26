package main

import (
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/spf13/pflag"
	funk "github.com/thoas/go-funk"

	"github.com/stretchr/testify/assert"
)

// Little hack to ensure we are ran before the init function in state.go
var _ = func() (_ struct{}) {
	config.Init()
	locale.Init()
	return
}()

func TestInit(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	pflags := Cc.PersistentFlags()
	var flags []string

	pflags.VisitAll(func(pf *pflag.Flag) {
		flags = append(flags, pf.Name)
	})

	assert.Equal(funk.Contains(flags, "locale"), true, "locale pflag is set")
}

func TestMainFn(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--help"})

	main()

	assert.Equal(true, true, "main didn't panic")
}

func TestMainError(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--foo"})

	exitCode := 0
	exit = func(code int) {
		exitCode = 1
	}

	main()

	assert.Equal(exitCode, 1, "main didn't exit")
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	Execute(Cc, []string{"--help"})

	assert.Equal(true, true, "Execute didn't panic")
}
