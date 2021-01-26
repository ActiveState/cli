package captain

import (
	"errors"
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

type PackageVersion struct {
	NameVersion
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.NewError("err_package_format", "The package and version provided is not formatting correctly, must be in the form of <package>@<version>")
	}
	return nil
}

var _ FlagMarshaler = &StateToolChannelVersion{}

type StateToolChannelVersion struct {
	NameVersion
}

func (stv *StateToolChannelVersion) Set(arg string) error {
	err := stv.NameVersion.Set(arg)
	if err != nil {
		return locale.NewError("err_channel_format", "The State Tool channel and version provided is not formatting correctly, must be in the form of <channel>@<version>")
	}
	return nil
}

func (stv *StateToolChannelVersion) Type() string {
	return "channel"
}

func (stv *StateToolChannelVersion) String() string {
	return stv.name
}

type PlatformVersion struct {
	NameVersion
}

func (pv *PlatformVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.NewError("err_platform_format", "The platform and version provided is not formatting correctly, must be in the form of <platform>@<version>")
	}
	return nil
}
