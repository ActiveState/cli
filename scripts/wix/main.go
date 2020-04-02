package main

import (
	"errors"
	"log"
	"os"
	"text/template"

	"github.com/ActiveState/cli/internal/constants"
)

type Product struct {
	Version string
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("Must provide only filepath to .wxs file"))
	}
	wxsFile := os.Args[1]

	file, err := os.OpenFile(wxsFile, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	tmpl := template.Must(template.ParseFiles(wxsFile))
	data := Product{constants.VersionNumber}
	file.Truncate(0)
	file.Seek(0, 0)
	err = tmpl.Execute(file, data)
	if err != nil {
		log.Fatal(err)
	}
}
