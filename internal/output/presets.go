package output

import (
	"fmt"
)

type Title string

func (t Title) String() string {
	return fmt.Sprintf("[HEADING]█ %s[/RESET]\n", string(t))
}

func (t Title) MarshalOutput(f Format) interface{} {
	return t.String()
}

func (t Title) MarshalStructured(f Format) interface{} {
	return Suppress
}

type Emphasize string

func (h Emphasize) String() string {
	return fmt.Sprintf("\n[HEADING]%s[/RESET]", string(h))
}

func (h Emphasize) MarshalOutput(f Format) interface{} {
	return h.String()
}

func (h Emphasize) MarshalStructured(f Format) interface{} {
	return Suppress
}

type plainOutput struct {
	plain interface{}
}

type structuredOutput struct {
	structured interface{}
}

type preparedOutput struct {
	*plainOutput
	*structuredOutput
}

func (o *plainOutput) MarshalOutput(_ Format) interface{} {
	return o.plain
}

func (o *structuredOutput) MarshalStructured(_ Format) interface{} {
	return o.structured
}

func Prepare(plain interface{}, structured interface{}) *preparedOutput {
	return &preparedOutput{&plainOutput{plain}, &structuredOutput{structured}}
}

func Structured(structured interface{}) *structuredOutput {
	return &structuredOutput{structured}
}

