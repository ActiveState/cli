// Unit tests for the logging module

package logging

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"
)

var _ = regexp.Compile

type Test1Handler struct {
	formatter Formatter
	file      *os.File
	messages  []string
}

func (l *Test1Handler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *Test1Handler) SetVerbose(v bool) {
}

func (l *Test1Handler) SetConfig(cfg config) {}

func (l *Test1Handler) Output() io.Writer {
	return nil
}

func (l *Test1Handler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	l.messages = append(l.messages, fmt.Sprintf("%s", message))

	return nil
}

func (l *Test1Handler) Printf(msg string, args ...interface{}) {}

func (l *Test1Handler) Reset() {
	l.messages = make([]string, 0)
}

func (l *Test1Handler) Len() int {
	return len(l.messages)
}

func (l *Test1Handler) Close() {}

func logAllLevels(msg string) {
	Debug(msg)
	Info(msg)
	Warning(msg)
	Error(msg)
	//we do not test critical as it calls GoExit
	//Critical(msg)
}

func Test_SetLevelByString(t *testing.T) {

	w := &Test1Handler{DefaultFormatter, nil, nil}
	SetHandler(w)

	w.Reset()
	//test levels
	e := SetMinimalLevelByName("DEBUG")
	fmt.Println(level)
	if e != nil {
		t.Fatalf("Could not set level by name: %s", e)
	}
	logAllLevels("Hello world")

	if w.Len() != 4 {
		t.Errorf("Not all levels logged - got  %d messages", w.Len())
	}
	e = SetMinimalLevelByName("ERROR")
	if e != nil {
		t.Fatalf("Could not set level by name: %s", e)
	}
	w.Reset()
	logAllLevels("Hello world")
	if w.Len() != 1 {
		t.Errorf("Not all levels logged - got  %d messages", w.Len())
	}

	e = SetMinimalLevelByName("FOOBAR")
	if e == nil {
		t.Fatalf("This should have raised an error...")
	}

}
func Test_Logging(t *testing.T) {

	w := &Test1Handler{DefaultFormatter, nil, nil}
	SetHandler(w)

	w.Reset()
	//test levels
	SetLevel(0)
	logAllLevels("Hello world")

	if w.Len() > 0 {
		t.Errorf("Got messages for level 0")
	}

	SetLevel(ALL)
	w.Reset()
	logAllLevels("Hello world")
	fmt.Printf("Received %d messages\n", w.Len())
	if w.Len() != 4 {
		fmt.Println(w.messages)
		t.Errorf("Did not log all errors (%d)", w.Len())
	}

	levels := []int{DEBUG, INFO, WARNING, ERROR}

	for l := range levels {

		w.Reset()
		SetLevel(levels[l])
		logAllLevels("Testing")

		if !(w.Len() == 1 || (levels[l] == CRITICAL && w.Len() == 2)) {
			t.Errorf("Wrong number of messages written: %d. level: %d", w.Len(), levels[l])
		}
	}

}

type TestHandler struct {
	output    [][]interface{}
	formatter Formatter
	t         *testing.T
}

func (t *TestHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	t.output = append(t.output, []interface{}{ctx.Level, ctx.File, message, ctx.Line, args})
	fmt.Println(*ctx)
	if ctx.Line <= 0 || ctx.Level == "" {
		t.t.Fatalf("Invalid args")
	}
	return nil
}

func (t *TestHandler) SetFormatter(fmt Formatter) {
	t.formatter = fmt
}

func (l *TestHandler) Output() io.Writer {
	return nil
}

func (l *TestHandler) SetVerbose(v bool) {
}

func (l *TestHandler) SetConfig(cfg config) {}

func (l *TestHandler) Printf(msg string, args ...interface{}) {
}

func (l *TestHandler) Close() {}

func Test_Handler(t *testing.T) {

	handler := &TestHandler{
		make([][]interface{}, 0),
		DefaultFormatter,
		t,
	}
	SetHandler(handler)

	SetLevel(ALL)
	Info("Foo Bar %s", 1)
	Warning("Bar Baz %s", 2)

	if len(handler.output) != 2 {
		t.Fatal("Wrong len of output handler ")
	}

	fmt.Println("Passed testHandler")
}
func Test_Formatting(t *testing.T) {
	ctx := &MessageContext{
		Level: "TEST",
		File:  "testung",
		Line:  100,
	}

	formatter := DefaultFormatter

	msg := formatter.Format(ctx, "FOO %s", "bar")

	//fmt.Println("Message: ", msg)
	if msg != "[TEST Jan  1 00:00:00.000000000, testung:100] FOO bar" {
		t.Fatal("Got wrong formatting:", msg)
	}

	//"[%[1]s %[2]s, %[3]s:%[4]d] %[5]s",
	format := "%[5]s @ %[4]d:%[3]s: %[2]s %[1]s"
	formatter = &SimpleFormatter{format}

	mesg := "FOO %s"

	s := formatter.Format(ctx, mesg, "BAR")
	fmt.Println(s)
	if s != "FOO BAR @ 100:testung: Jan  1 00:00:00.000000000 TEST" {
		t.Fatal(s)
	}

}
