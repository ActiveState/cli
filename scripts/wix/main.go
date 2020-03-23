package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("Must provide only filepath to .wxs file"))
	}

	xmlRaw, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	versionRE := regexp.MustCompile(`REPLACE-VERSION`)
	updated := versionRE.ReplaceAll(xmlRaw, []byte(constants.VersionNumber))

	err = ioutil.WriteFile(os.Args[1], updated, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
