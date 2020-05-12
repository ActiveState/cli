package output

type Mediator struct {
	Outputer
	format Format
}

type Marshaller interface {
	MarshalOutput(Format) interface{}
}

func (m *Mediator) Print(v interface{}) {
	m.Outputer.Print(mediatorValue(v, m.format))
}

func (m *Mediator) Error(v interface{}) {
	m.Outputer.Error(mediatorValue(v, m.format))
}

func (m *Mediator) Notice(v interface{}) {
	m.Outputer.Notice(mediatorValue(v, m.format))
}

func mediatorValue(v interface{}, format Format) interface{} {
	vt, ok := v.(Marshaller)
	if !ok {
		return v
	}
	return vt.MarshalOutput(format)
}
