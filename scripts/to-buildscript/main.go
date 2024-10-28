package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/go-openapi/strfmt"
)

func main() {
	var input string
	var argOffset = 0
	if len(os.Args) == 2 && fileutils.FileExists(os.Args[1]) {
		input = string(fileutils.ReadFileUnsafe(os.Args[1]))
		argOffset = 1
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if errors.Is(scanner.Err(), bufio.ErrFinalToken) {
				break
			}
			line := scanner.Text()
			if line == "\x04" { // Ctrl+D character
				break
			}
			input += line + "\n"
		}

		if err := scanner.Err(); err != nil {
			panic(fmt.Sprintf("error reading standard input: %v\n", err))
		}
	}

	if input == "" {
		fmt.Printf("Usage: %s [<< <buildexpression-blob> | <filepath>] [<timestamp>]\n", os.Args[0])
		os.Exit(1)
	}

	project := "https://platform.activestate.com/org/project?branch=main&commitID=00000000-0000-0000-0000-000000000000"
	var atTime *time.Time
	if len(os.Args) == (2 + argOffset) {
		t, err := time.Parse(strfmt.RFC3339Millis, os.Args[1+argOffset])
		if err != nil {
			panic(errs.JoinMessage(err))
		}
		atTime = &t
	}

	bs := buildscript.New()
	bs.SetProject(project)
	if atTime != nil {
		bs.SetAtTime(*atTime, true)
	}
	err := bs.UnmarshalBuildExpression([]byte(input))
	if err != nil {
		panic(errs.JoinMessage(err))
	}
	b, err := bs.Marshal()
	if err != nil {
		panic(errs.JoinMessage(err))
	}

	fmt.Println(string(b))
}
