package packages

import (
	"fmt"
	"strings"
)

type PackageVersion struct {
	Name    string
	Version string
}

func (pv *PackageVersion) Set(arg string) error {
	nameArg := strings.Split(arg, "@")
	pv.Name = nameArg[0]
	if len(nameArg) == 2 {
		pv.Version = nameArg[1]
	}
	return nil
}

func (pv *PackageVersion) String() string {
	if pv.Version == "" {
		return pv.Name
	}
	return fmt.Sprintf("%s@%s", pv.Name, pv.Version)
}
