package secrethelper

import (
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
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
