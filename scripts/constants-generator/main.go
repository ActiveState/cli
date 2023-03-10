package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/dave/jennifer/jen"

	"github.com/ActiveState/cli/internal/constants/preprocess"
)

var inTest = false

var (
	start = time.Now()
)

func log(format string, as ...any) {
	d := time.Since(start)
	fmt.Fprintf(os.Stderr, "[%08dμs] ", d.Microseconds())
	fmt.Fprintf(os.Stderr, format, as...)
	fmt.Fprintln(os.Stderr)
}

func logFatal(format string, as ...any) {
	log(format, as...)
	os.Exit(1)
}

func main() {
	if !inTest {
		run(os.Args)
	}
}

func run(args []string) {
	log("Starting run logic")
	if len(args) < 2 || (args[1] == "--" && len(args) < 3) {
		logFatal("Usage: %s <target-file>", args[0])
	}

	log("Storing constants data in file buffer")
	f := jen.NewFile("constants")
	f.HeaderComment("Do NOT manually edit this file. It is generated using scripts/constants-generator using data from constants/preprocess.")
	f.HeaderComment("This script is automatically ran by `make build`, so you shouldn't have to run this manually.")

	for k, v := range preprocess.Constants {
		log("Adding Constant: %s, Value: %s", k, v())
		f.Const().Id(k).Op("=").Lit(v())
	}

	log("Rendering file buffer")
	buf := &bytes.Buffer{}
	err := f.Render(buf)
	if err != nil {
		logFatal("Rendering failed: %v", err)
	}

	target := args[1]
	if target == "--" {
		target = args[2]
	}

	wd, _ := os.Getwd()
	log("Writing generated constants to: %s (pwd: %s)", target, wd)
	ioutil.WriteFile(target, buf.Bytes(), 0666)

	log("Constants file generated")
}
