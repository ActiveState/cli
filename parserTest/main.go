package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/buildexpression/parser"
)

func main() {
	err := runParser()
	if err != nil {
		fmt.Println(errs.JoinMessage(err))
		os.Exit(1)
	}
}

func runLexer() error {
	root, err := environment.GetRootPath()
	if err != nil {
		return errs.Wrap(err, "Could not get current working directory")
	}

	testData, err := fileutils.ReadFile(filepath.Join(root, "pkg", "buildexpression", "testdata", "buildexpression.json"))
	if err != nil {
		return errs.Wrap(err, "Could not read lexer test data")
	}
	l := parser.NewLexer(testData)

	for {
		pos, token, lit, err := l.Scan()
		if err != nil {
			return err
		}
		if token == parser.EOF {
			break
		}
		fmt.Printf("%s:%d:%d %s %s\n", "buildexpression.json", pos.Line, pos.Column, token, lit)
	}

	return nil
}

func runParser() error {
	wd, err := environment.GetRootPath()
	if err != nil {
		return errs.Wrap(err, "Could not get current working directory")
	}

	testData, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildexpression", "testdata", "buildexpression.json"))
	if err != nil {
		return errs.Wrap(err, "Could not read lexer test data")
	}

	p := parser.New(testData)
	tree, err := p.Parse()
	if err != nil {
		return errs.Wrap(err, "Could not parse buildexpression.json")
	}

	tree.Walk(func(node *parser.Node) error {
		fmt.Printf("%s:%d:%d %s %s\n", "buildexpression.json", node.Position().Line, node.Position().Column, node.Type(), node.Literal())
		return nil
	})

	return nil
}
