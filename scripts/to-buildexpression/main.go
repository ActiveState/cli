package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript"
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
		fmt.Printf("Usage: %s << <buildscript-blob>\n", os.Args[0])
		os.Exit(1)
	}

	bs, err := buildscript.Unmarshal([]byte(input))
	if err != nil {
		panic(errs.JoinMessage(err))
	}
	b, err := bs.MarshalBuildExpression()
	if err != nil {
		panic(errs.JoinMessage(err))
	}
	fmt.Println(string(b))
}