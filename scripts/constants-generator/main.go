package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dave/jennifer/jen"

	"github.com/ActiveState/cli/internal/constants/preprocess"
)

var (
	inTest  bool
	start   = time.Now()
	verbose bool
	silent  bool
)

func logit(format string, as ...any) {
	if verbose {
		d := time.Since(start)
		fmt.Fprintf(os.Stderr, "[%08dμs] ", d.Microseconds())
	}
	fmt.Fprintf(os.Stderr, format, as...)
	fmt.Fprintln(os.Stderr)
}

func log(format string, as ...any) {
	if silent {
		return
	}
	logit(format, as...)
}

func logFatal(format string, as ...any) {
	logit(format, as...)
	os.Exit(1)
}

func main() {
	flag.BoolVar(&verbose, "v", verbose, "Turn on verbose output")
	flag.BoolVar(&silent, "s", silent, "Silence all non-critical output")
	flag.Parse()

	if !inTest {
		run(os.Args)
	}
}

func run(args []string) {
	log("Starting run logic")
	if len(args) < 2 || strings.HasPrefix(args[len(args)-1], "-") {
		logFatal("Usage: %s [-v] <target-file>", args[0])
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

	target := args[len(args)-1]

	wd, _ := os.Getwd()
	log("Writing generated constants to: %s (pwd: %s)", target, wd)
	os.WriteFile(target, buf.Bytes(), 0666)

	log("Constants file generated")
}
