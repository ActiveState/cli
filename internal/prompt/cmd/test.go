package main

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics/client/async"
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
	p := prompt.New(true, async.New())

	selectDefault := "choice 1"
	p.Select("Select", "Please select one", []string{"choice 1", "choice 2", "choice 3"}, &selectDefault)

	confirmDefault := true
	p.Confirm("Confirm", "Do you confirm?", &confirmDefault)

	inputDefault := "Default response"
	p.Input("Input with Default", "Write something or use default", &inputDefault)

	inputDefault = ""
	p.Input("Input", "Please write something", &inputDefault)

	p.Input("Input", "Please write something, again", new(string))

	p.InputSecret("Secret", "Please write something secretive")
}
