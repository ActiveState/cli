package main

import (
	"testing"

	"github.com/spf13/pflag"
	funk "github.com/thoas/go-funk"

	"github.com/stretchr/testify/assert"
)

func Testinit(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	pflags := Cc.PersistentFlags()
	var flags []string

	pflags.VisitAll(func(pf *pflag.Flag) {
		flags = append(flags, pf.Name)
	})

	assert.Equal(funk.Contains(flags, "locale"), true, "locale pflag is set")
}

func TestMain(t *testing.T) {
	t.Skip("not implemented yet")
}
func TestExecute(t *testing.T) {
	t.Skip("not implemented yet")
}
