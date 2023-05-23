package buildscript

import (
	"bytes"
	"io"
)

func (s *Script) ToJson() []byte {
	json := &bytes.Buffer{}
	s.WriteJson(json)
	return json.Bytes()
}

func writeString(w io.Writer, s string) {
	w.Write([]byte(`"`))
	w.Write([]byte(s))
	w.Write([]byte(`"`))
}

func (s *Script) WriteJson(w io.Writer) {
	w.Write([]byte(`{"let":{`))
	for i, assignment := range s.Let.Assignments {
		assignment.WriteJson(w)
		if i+1 < (len(s.Let.Assignments)) {
			w.Write([]byte(","))
		}
	}
	w.Write([]byte(`,"in":`))
	switch {
	case s.In.FuncCall != nil:
		s.In.FuncCall.WriteJson(w)
	case s.In.Name != nil:
		writeString(w, "$"+*s.In.Name)
	}
	w.Write([]byte(`}}`))
}

func (a *Assignment) WriteJson(w io.Writer) {
	writeString(w, a.Key)
	w.Write([]byte(":"))
	a.Value.WriteJson(w)
}

func (v *Value) WriteJson(w io.Writer) {
	switch {
	case v.FuncCall != nil:
		v.FuncCall.WriteJson(w)

	case v.List != nil:
		w.Write([]byte("["))
		for i, item := range *v.List {
			item.WriteJson(w)
			if i+1 < len(*v.List) {
				w.Write([]byte(","))
			}
		}
		w.Write([]byte("]"))

	case v.Str != nil:
		w.Write([]byte(*v.Str))

	case v.Assignment != nil:
		v.Assignment.WriteJson(w)

	case v.Object != nil:
		w.Write([]byte("{"))
		for i, pair := range *v.Object {
			pair.WriteJson(w)
			if i+1 < len(*v.Object) {
				w.Write([]byte(","))
			}
		}
		w.Write([]byte("}"))

	case v.Ident != nil:
		writeString(w, *v.Ident)
	}
}

func (f *FuncCall) WriteJson(w io.Writer) {
	w.Write([]byte("{"))
	writeString(w, f.Name)
	w.Write([]byte(":{"))
	for i, argument := range f.Arguments {
		argument.WriteJson(w)
		if i+1 < len(f.Arguments) {
			w.Write([]byte(","))
		}
	}
	w.Write([]byte("}}"))
}
