package captain

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

type NameVersion struct {
	name    string
	version string
}

func (nv *NameVersion) Set(arg string) error {
	nameArg := strings.Split(arg, "@")
	nv.name = nameArg[0]
	if len(nameArg) == 2 {
		nv.version = nameArg[1]
	}
	if len(nameArg) > 2 {
		return locale.NewInputError("name_version_format_err", "Invalid format: Should be <name@version>")
	}
	return nil
}

func (nv *NameVersion) String() string {
	if nv.version == "" {
		return nv.name
	}
	return fmt.Sprintf("%s@%s", nv.name, nv.version)
}

func (nv *NameVersion) Name() string {
	return nv.name
}

func (nv *NameVersion) Version() string {
	return nv.version
}
