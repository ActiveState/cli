package installCmd

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
)

type cmdOpts struct{}

func (c *cmdOpts) Execute(args []string) error {
	T := locale.T
	print.Line(T("running_install"))
	return nil
}

var cmd cmdOpts

// Register the install command
func Register() (name string, shortDescription string, longDescription string, data interface{}) {
	T := locale.T
	return "install", T("install_project"), T("install_project_long"), &cmd
}
