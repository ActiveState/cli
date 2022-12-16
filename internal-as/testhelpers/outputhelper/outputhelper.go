package outputhelper

import (
	"bytes"
	"fmt"

	"github.com/ActiveState/cli/internal/output"
)

type Catcher struct {
	output.Outputer
	cfg       *output.Config
	outWriter *bytes.Buffer
	errWriter *bytes.Buffer
}

func NewCatcher() *Catcher {
	return NewCatcherByFormat(output.PlainFormatName, false)
}

func NewCatcherByFormat(format output.Format, interactive bool) *Catcher {
	catch := &Catcher{}

	catch.cfg = &output.Config{
		Colored:     false,
		Interactive: interactive,
	}

	catch.outWriter = &bytes.Buffer{}
	catch.errWriter = &bytes.Buffer{}

	outputer, err := output.New(string(format), &output.Config{
		OutWriter:   catch.outWriter,
		ErrWriter:   catch.errWriter,
		Colored:     false,
		Interactive: false,
	})

	if err != nil {
		panic(fmt.Sprintf("Could not create plain outputer: %s", err.Error()))
	}

	catch.Outputer = outputer

	return catch
}

func (c *Catcher) Output() string {
	return c.outWriter.String()
}

func (c *Catcher) ErrorOutput() string {
	return c.errWriter.String()
}

func (c *Catcher) CombinedOutput() string {
	return c.Output() + "\n" + c.ErrorOutput()
}

type TypedCatcher struct {
	Prints  []interface{}
	Errors  []interface{}
	Notices []interface{}
}

func (t *TypedCatcher) Type() output.Format {
	return ""
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
