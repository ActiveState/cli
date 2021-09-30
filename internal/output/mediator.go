package output

import "io"

type Mediator struct {
	Outputer
	format Format
}

type Marshaller interface {
	MarshalOutput(Format) interface{}
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
	vt, ok := v.(Marshaller)
	if !ok {
		return v
	}
	return vt.MarshalOutput(format)
}

// MediatedFormatter provides a custom type that can be used to conveniently create different outputs for different formats
// eg. `NewFormatter("Hello John!").WithFormat(JSONFormatName, "John")`
// This would print "Hello John!" with the plain formatter and just "John" with the JSON formatter
type MediatedFormatter struct {
	formatters map[Format]interface{}
	output     interface{}
}

func NewFormatter(defaultOutput interface{}) MediatedFormatter {
	return MediatedFormatter{map[Format]interface{}{}, defaultOutput}
}

func (m MediatedFormatter) WithFormat(format Format, output interface{}) MediatedFormatter {
	m.formatters[format] = output
	return m
}

func (m MediatedFormatter) MarshalOutput(format Format) interface{} {
	if v, ok := m.formatters[format]; ok {
		return v
	} else {
		if format == EditorFormatName || format == EditorV0FormatName {
			return m.MarshalOutput(JSONFormatName) // fall back on JSON
		}
	}
	return m.output
}
