package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/go-openapi/strfmt"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var input string
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

	if input == "" {
		fmt.Printf("Usage: %s [<at-time>] << <buildexpression-blob>\n", os.Args[0])
		os.Exit(1)
	}

	var atTime *time.Time
	if len(os.Args) == 2 {
		t, err := time.Parse(strfmt.RFC3339Millis, os.Args[1])
		if err != nil {
			panic(errs.JoinMessage(err))
		}
		atTime = &t
	}

	bs, err := buildscript.UnmarshalBuildExpression([]byte(input), atTime)
	if err != nil {
		panic(errs.JoinMessage(err))
	}
	b, err := bs.Marshal()
	if err != nil {
		panic(errs.JoinMessage(err))
	}

	fmt.Println(string(b))
}
