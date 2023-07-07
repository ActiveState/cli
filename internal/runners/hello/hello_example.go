package hello

import (
	"fmt"

	"github.com/ActiveState/cli/internal/primer"
)

type primeable interface {
	primer.Projecter
}

type RunParams struct {
	Named string
}

func NewRunParams() *RunParams {
	return &RunParams{
		Named: "Friend",
	}
}

type Hello struct{}

func New(p primeable) *Hello {
	return &Hello{}
}

func (h *Hello) Run(params *RunParams) error {
	fmt.Printf("hello, %s!\n", params.Named)
	return nil
}
