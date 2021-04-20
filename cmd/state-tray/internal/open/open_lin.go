//+build linux

package open

import "errors"

func Prompt(command string) error {
	return errors.New("not implemented")
}

func getPrompt() string {
	panic(errors.New("not implemented"))
}
