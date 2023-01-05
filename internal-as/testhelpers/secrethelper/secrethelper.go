package secrethelper

import (
	"strings"

	"github.com/ActiveState/cli/internal-as/environment"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/exeutils"
)

func GetSecretIfEmpty(value string, key string) string {
	if value != "" {
		return value
	}
	out, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", []string{"secrets", "get", key}, []string{})
	if err != nil {
		panic(errs.Wrap(err, stderr))
	}
	return strings.TrimSpace(out)
}

func GetSecret(key string) string {
	out, stderr, err := exeutils.ExecSimpleFromDir(environment.GetRootPathUnsafe(), "state", []string{"secrets", "get", key}, []string{})
	if err != nil {
		panic(errs.Wrap(err, stderr))
	}
	return strings.TrimSpace(out)
}
