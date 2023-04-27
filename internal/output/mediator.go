package output

import (
	"io"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
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

func mediatorValue(v interface{}, format Format) interface{} {
	if format.IsStructured() {
		if vt, ok := v.(StructuredMarshaller); ok {
			return vt.MarshalStructured(format)
		}
		multilog.Error("%s output not supported for message: %v", string(format), v)
		return jsonError{
			[]string{locale.Tl(
				"err_no_structured_output",
				"This command does not support the {{.V0}} output format. Please try again without that output flag",
				string(format),
			)},
			1,
		}
	}
	if vt, ok := v.(Marshaller); ok {
		return vt.MarshalOutput(format)
	}
	return v
}
