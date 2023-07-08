package runbits

import (
	"errors"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

func SayHello(out output.Outputer, to string) error {
	if to == "" {
		return errors.New("test")
	}

	out.Print(locale.Tl("hello_message", "Hello, {{.V0}}!", to))

	return nil
}
