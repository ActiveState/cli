package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dave/jennifer/jen"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants/preprocess"
)

func main() {
	if !condition.InUnitTest() {
		run(os.Args)
	}
}

func run(args []string) {
	if len(args) < 2 || (args[1] == "--" && len(args) < 3) {
		log.Fatalf("Usage: %s <target-file>", args[0])
	}

	f := jen.NewFile("constants")

	f.HeaderComment("Do NOT manually edit this file. It is generated using scripts/constants-generator using data from constants/preprocess.")
	f.HeaderComment("This script is automatically ran by `make build`, so you shouldn't have to run this manually.")

	for k, v := range preprocess.Constants {
		fmt.Printf("Adding Constant: %s, Value: %s\n", k, v())
		f.Const().Id(k).Op("=").Lit(v())
	}

	buf := &bytes.Buffer{}
	err := f.Render(buf)
	if err != nil {
		log.Fatalf("Rendering failed: %v", err)
	}

	target := args[1]
	if target == "--" {
		target = args[2]
	}

	wd, _ := os.Getwd()
	fmt.Printf("Writing generated constants to: %s (pwd: %s)\n", target, wd)
	ioutil.WriteFile(target, buf.Bytes(), 0666)

	fmt.Println("Constants generated")
}
