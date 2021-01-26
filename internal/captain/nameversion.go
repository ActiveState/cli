package captain

import (
	"errors"
	"fmt"
	"strings"
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
		return errors.New("invalid format")
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
