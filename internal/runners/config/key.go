package config

import (
	"regexp"

	"github.com/ActiveState/cli/internal/locale"
)

type Key string

// Set implements the captain ArgMarshaler interface
func (k Key) Set(v string) error {
	err := validateKey(v)
	if err != nil {
		return locale.WrapInputError(err, "err_set_invalid_key", "The config key can only consist of alphanumeric characters and a '.'")
	}

	k = Key(v)

	return nil
}

func (k Key) String() string {
	return string(k)
}

func validateKey(key string) error {
	regex := regexp.MustCompile(`^[A-Za-z0-9\.]+$`)
	if !regex.MatchString(key) {
		return locale.NewInputError("err_config_invalid_key", "Invalid config key")
	}
	return nil
}
