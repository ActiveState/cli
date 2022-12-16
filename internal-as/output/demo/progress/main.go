package main

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/output"
)

func main() {
	out, err := output.New(string(output.PlainFormatName), &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: true,
	})
	if err != nil {
		panic(err)
	}

	p := output.StartSpinner(out, "Demo is doing something", 100*time.Millisecond)
	time.Sleep(5 * time.Second)
	p.Stop("Done")
}
