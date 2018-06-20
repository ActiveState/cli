package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	//"io/ioutil"
	"log"
	//"path/filepath"

	"github.com/ActiveState/cli/internal/constants/preprocess"
	"github.com/ActiveState/cli/internal/environment"
	//"github.com/ActiveState/cli/internal/environment"

	"github.com/dave/jennifer/jen"
)

func main() {
	if flag.Lookup("test.v") == nil {
		run()
	}
}

func run() {
	f := jen.NewFile("constants")

	f.HeaderComment("Do NOT manually edit this file. It is generated using scripts/constants-generator using data from constants/preprocess.")
	f.HeaderComment("This script is automatically ran by `make build`, so you shouldn't have to run this manually.")

	for k, v := range preprocess.Constants {
		f.Const().Id(k).Op("=").Lit(v())
	}

	buf := &bytes.Buffer{}
	err := f.Render(buf)
	if err != nil {
		log.Fatalf("Rendering failed: %v", err)
	}

	target := filepath.Join(environment.GetRootPathUnsafe(), "internal", "constants", "generated.go")
	ioutil.WriteFile(target, buf.Bytes(), 0666)

	fmt.Println("Constants generated")
}
