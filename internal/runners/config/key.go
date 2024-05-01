package config

import (
	"regexp"

	"github.com/ActiveState/cli/internal/locale"
)

type Key string

// Set implements the captain ArgMarshaler interface
func (k *Key) Set(v string) error {
	regex := regexp.MustCompile(`^[A-Za-z0-9]+(\.[A-Za-z0-9_]+)*$`)
	if !regex.MatchString(v) {
		return locale.NewInputError("err_set_invalid_key", "Invalid config key. The config key can only consist of alphanumeric and dot characters")
	}

	*k = Key(v)

	return nil
}

func (k Key) String() string {
	return string(k)
}
