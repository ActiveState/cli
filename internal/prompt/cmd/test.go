package main

import (
	"os"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
)

func main() {
	output.New(string(output.PlainFormatName), &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: true,
	})
	p := prompt.New(true)

	p.Select("Select", "Please select one", []string{"choice 1", "choice 2", "choice 3"}, "choice 1")

	p.Confirm("Confirm", "Do you confirm?", true)

	p.Input("Input with Default", "Write something or use default", "Default response")

	p.Input("Input", "Please write something", "")

	p.InputSecret("Secret", "Please write something secretive")
}
