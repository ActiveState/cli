package output

import (
	"fmt"
	"io"

	"github.com/ActiveState/cli/internal/locale"
)

type Mediator struct {
	Outputer
	format Format
}

type Marshaller interface {
	MarshalOutput(Format) interface{}
}

type StructuredMarshaller interface {
	MarshalStructured(Format) interface{}
}

func (m *Mediator) Fprint(writer io.Writer, v interface{}) {
	if v = mediatorValue(v, m.format); v == Suppress {
		return
	}

	m.Outputer.Fprint(writer, v)
}

func (m *Mediator) Print(v interface{}) {
	if v = mediatorValue(v, m.format); v == Suppress {
		return
	}

	m.Outputer.Print(v)
}

func (m *Mediator) Error(v interface{}) {
	if v = mediatorValue(v, m.format); v == Suppress {
		return
	}

	m.Outputer.Error(v)
}

func (m *Mediator) Notice(v interface{}) {
	if v = mediatorValue(v, m.format); v == Suppress {
		return
	}

	m.Outputer.Notice(v)
}

func isStructuredFormat(format Format) bool {
	return format == JSONFormatName || format == EditorFormatName || format == EditorV0FormatName
}

func mediatorValue(v interface{}, format Format) interface{} {
	if isStructuredFormat(format) {
		if vt, ok := v.(StructuredMarshaller); ok {
			return vt.MarshalStructured(format)
		}
		strv := fmt.Sprintf("%v", v)
		return jsonError{[]string{locale.Tl("err_no_structured_output", "{{.V0}} output not supported for message: {{.V1}}", string(format), strv)}, 1}
	}
	if vt, ok := v.(Marshaller); ok {
		return vt.MarshalOutput(format)
	}
	return v
}
