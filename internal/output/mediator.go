package output

import (
	"io"

	"github.com/ActiveState/cli/internal/constants"
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

func mediatorValue(v interface{}, format Format) interface{} {
	if format.IsStructured() {
		if vt, ok := v.(StructuredMarshaller); ok {
			return vt.MarshalStructured(format)
		}
		return StructuredError{Message: locale.Tr("err_unsupported_structured_output", constants.ForumsURL)}
	}
	if vt, ok := v.(Marshaller); ok {
		return vt.MarshalOutput(format)
	}
	return v
}
