package hello

import (
	"fmt"

	"github.com/ActiveState/cli/internal/primer"
)

type infoPrimeable interface {
	primer.Projecter
}

type InfoRunParams struct {
	Extra bool
}

type Info struct{}

func NewInfo(p infoPrimeable) *Info {
	return &Info{}
}

func (i *Info) Run(params *InfoRunParams) error {
	fmt.Println("info")
	if params.Extra {
		fmt.Println("extra")
	}
	return nil
}
