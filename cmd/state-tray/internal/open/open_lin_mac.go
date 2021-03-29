//+build !windows

package open

import "errors"

func Prompt(command string) error {
	return errors.New("Not implemented")
}
