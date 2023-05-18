package buildscript

import (
	"bytes"
	"io"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

type Script struct {
	Let *Let `parser:"'let' ':' @@"`
	In  *In  `parser:"'in' ':' @@"`
}

type Let struct {
	Assignments []*Assignment `parser:"@@+"`
}

type Assignment struct {
	Key   string `parser:"@Ident '='"`
	Value *Value `parser:"@@"`
}

type Value struct {
	FuncCall *FuncCall `parser:"@@"`
	List     *[]*Value `parser:"| '[' @@ (',' @@)* ','? ']'"`
	String   *string   `parser:"| @String"`

	Assignment *Assignment    `parser:"| @@"`                        // only in FuncCall
	Object     *[]*Assignment `parser:"| '{' @@ (',' @@)* ','? '}'"` // only in List
	Ident      *string        `parser:"| @Ident"`                    // only in FuncCall
}

type FuncCall struct {
	Name      string   `parser:"@Ident"`
	Arguments []*Value `parser:"'(' @@ (',' @@)* ','? ')'"`
}

type In struct {
	FuncCall *FuncCall `parser:"@@"`
	Name     *string   `parser:"| @Ident"`
}

func (s *Script) Equals(other *model.BuildScript) bool { return false } // TODO

func (s *Script) Write(w io.Writer) {
	w.Write([]byte("let:\n"))
	for _, assignment := range s.Let.Assignments {
		assignment.Write(w, 1)
	}
	w.Write([]byte("\nin:\n\t"))
	switch {
	case s.In.FuncCall != nil:
		s.In.FuncCall.Write(w, 1)
	case s.In.Name != nil:
		w.Write([]byte(*s.In.Name))
	}
}

func (a *Assignment) Write(w io.Writer, indentLevel int) {
	w.Write(bytes.Repeat([]byte("\t"), indentLevel))
	w.Write([]byte(a.Key))
	w.Write([]byte(" = "))
	a.Value.Write(w, indentLevel)
}

func (v *Value) Write(w io.Writer, indentLevel int) {
	switch {
	case v.FuncCall != nil:
		v.FuncCall.Write(w, indentLevel)

	case v.List != nil:
		w.Write([]byte("[\n"))
		for i, item := range *v.List {
			if item.String != nil {
				w.Write(bytes.Repeat([]byte("\t"), indentLevel+1)) // string is on its own line, so indent
			}
			item.Write(w, indentLevel+1)
			if i+1 < len(*v.List) {
				w.Write([]byte(","))
			}
			w.Write([]byte("\n"))
		}
		w.Write(bytes.Repeat([]byte("\t"), indentLevel))
		w.Write([]byte("]"))

	case v.String != nil:
		w.Write([]byte(*v.String))

	case v.Assignment != nil:
		v.Assignment.Write(w, indentLevel)

	case v.Object != nil:
		w.Write(bytes.Repeat([]byte("\t"), indentLevel))
		w.Write([]byte("{\n"))
		for i, pair := range *v.Object {
			pair.Write(w, indentLevel+1)
			if i+1 < len(*v.Object) {
				w.Write([]byte(","))
			}
			w.Write([]byte("\n"))
		}
		w.Write(bytes.Repeat([]byte("\t"), indentLevel))
		w.Write([]byte("}"))

	case v.Ident != nil:
		w.Write([]byte(*v.Ident))
	}
}

func (f *FuncCall) Write(w io.Writer, indentLevel int) {
	w.Write([]byte(f.Name))
	w.Write([]byte("(\n"))
	for i, argument := range f.Arguments {
		argument.Write(w, indentLevel+1)
		if i+1 < len(f.Arguments) {
			w.Write([]byte(","))
		}
		w.Write([]byte("\n"))
	}
	w.Write(bytes.Repeat([]byte("\t"), indentLevel))
	w.Write([]byte(")\n"))
}
