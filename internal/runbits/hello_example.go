package runbits

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

func SayHello(out output.Outputer, name string) error {
	if name == "" {
		// Errors that are due to USER input should use `NewInputError` or `WrapInputError`
		return locale.NewInputError("hello_err_no_name", "No name provided.")
	}

	out.Print(locale.Tl("hello_message", "Hello, {{.V0}}!", name))

	return nil
}
