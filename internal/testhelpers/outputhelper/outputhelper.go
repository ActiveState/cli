package outputhelper

import (
	"bytes"
	"fmt"

	"github.com/ActiveState/cli/internal/output"
)

type catcher struct {
	Outputer  *output.Plain
	outWriter *bytes.Buffer
	errWriter *bytes.Buffer
}

func NewCatcher() *catcher {
	catch := &catcher{}

	catch.outWriter = &bytes.Buffer{}
	catch.errWriter = &bytes.Buffer{}

	outputer, fail := output.NewPlain(&output.Config{
		OutWriter:   catch.outWriter,
		ErrWriter:   catch.errWriter,
		Colored:     false,
		Interactive: false,
	})

	if fail != nil {
		panic(fmt.Sprintf("Could not create plain outputer: %s", fail.Error()))
	}

	catch.Outputer = &outputer

	return catch
}

func (c *catcher) Output() string {
	return c.outWriter.String()
}

func (c *catcher) ErrorOutput() string {
	return c.errWriter.String()
}

func (c *catcher) CombinedOutput() string {
	return c.Output() + "\n" + c.ErrorOutput()
}

type TypedCatcher struct {
	Prints  []interface{}
	Errors  []interface{}
	Notices []interface{}
}

func (t *TypedCatcher) Print(value interface{}) {
	t.Prints = append(t.Prints, value)
}

func (t *TypedCatcher) Error(value interface{}) {
	t.Errors = append(t.Errors, value)
}

func (t *TypedCatcher) Notice(value interface{}) {
	t.Notices = append(t.Notices, value)
}

func (t *TypedCatcher) Config() *output.Config {
	return &output.Config{}
}
