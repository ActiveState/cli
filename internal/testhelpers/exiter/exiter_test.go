package exiter_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/stretchr/testify/assert"
)

func TestWaitForExit(t *testing.T) {
	code := exiter.WaitForExit(func() {
		exiter.Exit(1)
	})
	assert.Equal(t, 1, code)
}

func TestWaitForExitNoExit(t *testing.T) {
	code := exiter.WaitForExit(func() {})
	assert.Equal(t, -1, code)
}

func TestWaitForExitPanic(t *testing.T) {
	assert.Panics(t, func() {
		code := exiter.WaitForExit(func() {
			panic("not exiter")
		})
		assert.Equal(t, -1, code)
	})
}
