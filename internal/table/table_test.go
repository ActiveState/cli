package table

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/output"
)

func TestStuff(t *testing.T) {
	out, fail := output.New(string(output.PlainFormatName), &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: true,
	})
	if fail != nil {
		t.Fatal(fail)
	}
	table := Create([][]string{{"some", "stuff"}})
	table.AddHeader([]string{"First", "Second"})
	table.WithHeaderFormat("[HEADING]%s[/RESET]")
	render, err := table.Render()
	if err != nil {
		t.Fatal(err)
	}
	out.Print(render)
}
