package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/alecthomas/participle/v2"
)

type Expression struct {
	Let *Let `parser:"'{' 'let' ':' @@ '}'"`
}

type Let struct {
	Assignments []*Assignment `parser:"'{' @@+ '}'"`
	In          *In           `parser:"| 'in' ':' @@"`
}

type Assignment struct {
	Key   string `parser:"@Ident ':'"`
	Value *Value `parser:"'{'? @@ '}'?"`
}

type Value struct {
	FuncCall *FuncCall `parser:"'{' @@ '}'"`
	List     *[]*Value `parser:"| '[' (@@ (',' @@)* ','?)? ']'"`
	Str      *string   `parser:"| @String"`
	Null     *Null     `parser:"| @@"`

	Assignment *Assignment    `parser:"| @@"`                             // only in FuncCall
	Object     *[]*Assignment `parser:"| '{' @@ (',' @@)* ','? '}' ','?"` // only in List
	Ident      *string        `parser:"| @Ident?"`                        // only in FuncCall
}

type Null struct {
	Null string `parser:"'null'"`
}

type FuncCall struct {
	Name      string   `parser:"@Ident ':' '{'"`
	Arguments []*Value `parser:"@@ (',' @@)* ','? '}'"`
}

type In struct {
	FuncCall *FuncCall `parser:"@@"`
	Name     *string   `parser:"| @Ident"`
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	wd, err := environment.GetRootPath()
	if err != nil {
		return err
	}

	data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "buildplanner", "model", "testdata", "buildexpression.json"))
	if err != nil {
		return err
	}

	cleaned := cleanJSON(string(data))
	parser, err := participle.Build[Expression]()
	if err != nil {
		return errs.Wrap(err, "Could not create parser for build script")
	}

	fmt.Println("EBNF:", parser.String())

	expr, err := parser.ParseBytes("test", []byte(cleaned), participle.Trace(os.Stdout))
	if err != nil {
		fmt.Println("Err: ", err)
		return errs.Wrap(err, "Could not parse build script")
	}

	fmt.Printf("Parsed expression: %+v\n", expr)

	return nil
}

func cleanJSON(s string) string {
	// Remove quotes around keys
	s = regexp.MustCompile(`"(\w+)":`).ReplaceAllString(s, "$1:")
	// Return the cleaned string
	return s
}

func oldRun() error {
	wd, err := environment.GetRootPath()
	if err != nil {
		return err
	}

	data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "buildplanner", "model", "testdata", "buildexpression.json"))
	if err != nil {
		return err
	}

	expr, err := model.NewBuildExpression2(data)
	if err != nil {
		return err
	}

	for _, let := range expr.Lets {
		fmt.Printf("Let arguments: %s\n", let.Arguments)
		fmt.Printf("Let in expression: %s\n", let.InExpr)
	}

	for _, ap := range expr.Aps {
		fmt.Printf("Ap name: %s\n", ap.Name)
		fmt.Printf("Ap arguments: %s\n", ap.Arguments)
	}

	for _, v := range expr.Vars {
		fmt.Printf("Var name: %s\n", v.Name)
		fmt.Printf("Var value: %s\n", v.Value)
	}

	return nil
}
