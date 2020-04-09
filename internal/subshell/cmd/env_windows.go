package cmd

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

func notExistError() error {
	return registry.ErrNotExist
}

func OpenKey(path string) (RegistryKey, error) {
	return registry.OpenKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
}

func IsNotExistError(err error) bool {
	return errors.Is(err, registry.ErrNotExist)
}