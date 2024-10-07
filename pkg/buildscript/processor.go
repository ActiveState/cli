package buildscript

type FuncProcessor interface {
	FuncName() string
	ToBuildExpression(*BuildScript, *FuncCall) error
	FromBuildExpression(*BuildScript, *FuncCall) error
}

type FuncProcessorMap map[string][]FuncProcessor

var DefaultProcessors FuncProcessorMap

func init() {
	DefaultProcessors = make(FuncProcessorMap)
}

func RegisterDefaultProcessor(marshaler FuncProcessor) {
	name := marshaler.FuncName()
	if _, ok := DefaultProcessors[name]; !ok {
		DefaultProcessors[name] = []FuncProcessor{}
	}
	DefaultProcessors[name] = append(DefaultProcessors[name], marshaler)
}

// RegisterProcessor registers a buildexpression marshaler for a buildscript function.
// Marshalers accept a buildscript Value, and marshals it to buildexpression JSON (e.g. an object).
// This is mainly (if not ONLY) used by tests, because for our main business logic we use the DefaultProcessors.
func (b *BuildScript) RegisterProcessor(name string, marshaler FuncProcessor) {
	if _, ok := b.processors[name]; !ok {
		b.processors[name] = []FuncProcessor{}
	}
	b.processors[name] = append(b.processors[name], marshaler)
}
