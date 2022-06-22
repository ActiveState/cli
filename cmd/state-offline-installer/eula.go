package main

import (
	"github.com/ActiveState/cli/internal/prompt"
	"io"
	"os"
)

func showEULAAndGetAcceptance(p *prompt.Prompter, filename string) (bool, error) {
	defaultChoice := "y"

	for {
		entered, err := (*p).Select("Setup", "Do you accept the Activestate End User Agreement? (d to display) [y/n/d]:", []string{"Y", "n", "d"}, &defaultChoice)
		// entered, err := getChar()
		if err != nil {
			return false, err
		}

		switch entered {
		case "y", "Y":
			return true, nil
		case "n", "N":
			return false, nil
		case "d", "D":
			f, err := os.Open(filename)
			if err != nil {
				return false, err
			}
			defer f.Close()

			// FIX: This isn't really going to fly, need to maybe consult with LE about how to do this
			_, err = io.Copy(os.Stdout, f)
			if err != nil {
				return false, err
			}
		}
	}
}
